// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

// This file contains the implementation of Go select statements.

import (
	"runtime/internal/atomic"
	"unsafe"
)

const debugSelect = false

// scase.kind values.
// Known to compiler.
// Changes here must also be made in src/cmd/compile/internal/gc/select.go's walkselectcases.
const (
	// scase类型的所有枚举值
	caseNil     = iota // nil通道
	caseRecv           // 从通道读取数据
	caseSend           // 发送数据至通道
	caseDefault        // default分支
)

// Select case descriptor.
// Known to compiler.
// Changes here must also be made in src/cmd/internal/gc/select.go's scasetype.
type scase struct {
	c           *hchan         // case语句中操作的通道
	elem        unsafe.Pointer // 指向数据地址，比如i <- ch中i变量的地址，ch<-j 中j变量的地址
	kind        uint16         // scase类型，见上面scase类型的所有枚举值
	pc          uintptr        // race pc (for race detector / msan)
	releasetime int64          // 用于pprof中记录select阻塞时间
}

var (
	chansendpc = funcPC(chansend)
	chanrecvpc = funcPC(chanrecv)
)

func selectsetpc(cas *scase) {
	cas.pc = getcallerpc()
}

func sellock(scases []scase, lockorder []uint16) {
	var c *hchan
	for _, o := range lockorder {
		c0 := scases[o].c
		// 由于lockorder按照通道的地址排序过，如果scase的通道是同一个，那么对应到lockorder中一定相邻的
		// 为了避免对同一个通道上锁两次（Go中锁不是可重入类型的锁），需要确保待上锁通道和上一次上锁的通道不是同一个(c0 != c)
		if c0 != nil && c0 != c {
			c = c0
			lock(&c.lock)
		}
	}
}

func selunlock(scases []scase, lockorder []uint16) {
	// We must be very careful here to not touch sel after we have unlocked
	// the last lock, because sel can be freed right after the last unlock.
	// Consider the following situation.
	// First M calls runtime·park() in runtime·selectgo() passing the sel.
	// Once runtime·park() has unlocked the last lock, another M makes
	// the G that calls select runnable again and schedules it for execution.
	// When the G runs on another M, it locks all the locks and frees sel.
	// Now if the first M touches sel, it will access freed memory.
	for i := len(scases) - 1; i >= 0; i-- {
		c := scases[lockorder[i]].c
		if c == nil {
			break
		}
		if i > 0 && c == scases[lockorder[i-1]].c {
			continue // will unlock it on the next iteration
		}
		unlock(&c.lock)
	}
}

func selparkcommit(gp *g, _ unsafe.Pointer) bool {
	// There are unlocked sudogs that point into gp's stack. Stack
	// copying must lock the channels of those sudogs.
	// Set activeStackChans here instead of before we try parking
	// because we could self-deadlock in stack growth on a
	// channel lock.
	gp.activeStackChans = true
	// Mark that it's safe for stack shrinking to occur now,
	// because any thread acquiring this G's stack for shrinking
	// is guaranteed to observe activeStackChans after this store.
	atomic.Store8(&gp.parkingOnChan, 0)
	// Make sure we unlock after setting activeStackChans and
	// unsetting parkingOnChan. The moment we unlock any of the
	// channel locks we risk gp getting readied by a channel operation
	// and so gp could continue running before everything before the
	// unlock is visible (even to gp itself).

	// This must not access gp's stack (see gopark). In
	// particular, it must not access the *hselect. That's okay,
	// because by the time this is called, gp.waiting has all
	// channels in lock order.
	var lastc *hchan
	for sg := gp.waiting; sg != nil; sg = sg.waitlink {
		if sg.c != lastc && lastc != nil {
			// As soon as we unlock the channel, fields in
			// any sudog with that channel may change,
			// including c and waitlink. Since multiple
			// sudogs may have the same channel, we unlock
			// only after we've passed the last instance
			// of a channel.
			unlock(&lastc.lock)
		}
		lastc = sg.c
	}
	if lastc != nil {
		unlock(&lastc.lock)
	}
	return true
}

/**
* 对于空select语句，会编译成block()，永远挂起当前G：
  select {
  }
*/
func block() {
	gopark(nil, nil, waitReasonSelectNoCases, traceEvGoStop, 1) // forever
}

/**
	select底层实现实现方式有4种：
	1. 对于单条case语句：
	select {
	case ch <- 1:
	}
	直接调用runtime.chansend1()或者runtime.chanrecv1()

	2. 对于空select
	select{
	}
	直接直接调用runtime.block()

	3. 对于两条分支语句，且其中一条为default语句（即通道非阻塞操作)
	3.1：
	select {
	case ch<-1:
    default:
	}
	上面语句将会编译成：
	if runtime.selectnbsend(c, v) {
		...
	} else {
		...
	}

	3.2：
	select {
	case <-ch:
    default:
	}
	上面语句将会编译成：
	if runtime.selectnbrecv(c, v) {
		...
	} else {
		...
	}

	4. 对于两条或两条以上case语句(两条语句时候，需不含default分支)
	select {
	case ch<-1:
	case ch<-2:
	case ch2<-3:
	}
	调用runtime.selectgo语句选择一个已就绪的分支执行(已就绪的含义就是当前case分支的通道可以读取或者写入)，若没有已就绪的分支，如果存在default分支，那么执行default分支
*/

// selectgo implements the select statement.
//
// cas0 points to an array of type [ncases]scase, and order0 points to
// an array of type [2*ncases]uint16. Both reside on the goroutine's
// stack (regardless of any escaping in selectgo).
//
// selectgo returns the index of the chosen scase, which matches the
// ordinal position of its respective select{recv,send,default} call.
// Also, if the chosen scase was a receive operation, it reports whether
// a value was received.
func selectgo(cas0 *scase, order0 *uint16, ncases int) (int, bool) {
	if debugSelect {
		print("select: cas0=", cas0, "\n")
	}

	// cas1指向[1<<16]scase，order1指向[1<<17]uint16
	// 数组尺寸需要使用标量指定，不能使用变量指定，所以通过cas0转换成cas1时候不能用(*[ncases]scase)(unsafe.Pointer(cas0))，
	// 为啥用1<<16，其实只要尺寸足够大就可以了，毕竟没有case分支语句能够达到2^16条的
	cas1 := (*[1 << 16]scase)(unsafe.Pointer(cas0))
	order1 := (*[1 << 17]uint16)(unsafe.Pointer(order0))

	scases := cas1[:ncases:ncases]               // scases是所有scase组成的切片（注意：Go语言中 数组是可以通过数组指针访问的）
	pollorder := order1[:ncases:ncases]          // pollorder是轮询序列，用于实现随机遍历scases切片
	lockorder := order1[ncases:][:ncases:ncases] // lockorder是上锁序列，用于实现对scase中的通道lock操作

	// Replace send/receive cases involving nil channels with
	// caseNil so logic below can assume non-nil channel.
	for i := range scases {
		cas := &scases[i]
		if cas.c == nil && cas.kind != caseDefault {
			// 将非default分支，且nil通道的case分支转换成kind为caseNil的scase
			// cas = &scase{c:nil, kind:caseNil}
			*cas = scase{}
		}
	}

	var t0 int64
	if blockprofilerate > 0 {
		t0 = cputicks()
		for i := 0; i < ncases; i++ {
			scases[i].releasetime = -1
		}
	}

	// The compiler rewrites selects that statically have
	// only 0 or 1 cases plus default into simpler constructs.
	// The only way we can end up with such small sel.ncase
	// values here is for a larger select in which most channels
	// have been nilled out. The general code handles those
	// cases correctly, and they are rare enough not to bother
	// optimizing (and needing to test).

	// generate permuted order
	// pollorder 是0到ncases的序列且已打乱
	for i := 1; i < ncases; i++ {
		j := fastrandn(uint32(i + 1))
		pollorder[i] = pollorder[j]
		pollorder[j] = uint16(i)
	}

	// sort the cases by Hchan address to get the locking order.
	// simple heap sort, to guarantee n log n time and constant stack footprint.
	// lockorder按照scase中通道的地址排序，这保证了具有相同通道的scase在lockerorder是相邻的，这么做是为了防止对同一个通道多次上锁，具体见sellock函数
	for i := 0; i < ncases; i++ {
		j := i
		// Start with the pollorder to permute cases on the same channel.
		c := scases[pollorder[i]].c
		for j > 0 && scases[lockorder[(j-1)/2]].c.sortkey() < c.sortkey() {
			k := (j - 1) / 2
			lockorder[j] = lockorder[k]
			j = k
		}
		lockorder[j] = pollorder[i]
	}
	for i := ncases - 1; i >= 0; i-- {
		o := lockorder[i]
		c := scases[o].c
		lockorder[i] = lockorder[0]
		j := 0
		for {
			k := j*2 + 1
			if k >= i {
				break
			}
			if k+1 < i && scases[lockorder[k]].c.sortkey() < scases[lockorder[k+1]].c.sortkey() {
				k++
			}
			if c.sortkey() < scases[lockorder[k]].c.sortkey() {
				lockorder[j] = lockorder[k]
				j = k
				continue
			}
			break
		}
		lockorder[j] = o
	}

	if debugSelect {
		for i := 0; i+1 < ncases; i++ {
			if scases[lockorder[i]].c.sortkey() > scases[lockorder[i+1]].c.sortkey() {
				print("i=", i, " x=", lockorder[i], " y=", lockorder[i+1], "\n")
				throw("select: broken sort")
			}
		}
	}

	// lock all the channels involved in the select
	sellock(scases, lockorder) // 对scases中所有通道上锁

	var (
		gp     *g
		sg     *sudog
		c      *hchan
		k      *scase
		sglist *sudog
		sgnext *sudog
		qp     unsafe.Pointer
		nextp  **sudog
	)

loop:
	// pass 1 - look for something already waiting
	var dfli int
	var dfl *scase
	var casi int
	var cas *scase
	var recvOK bool
	for i := 0; i < ncases; i++ { // 编译所有的case分支
		casi = int(pollorder[i])
		cas = &scases[casi]
		c = cas.c

		switch cas.kind {
		case caseNil: // nil通道，读或者写nil通道永远都是阻塞的，即该case永远不会选中selected
			continue

		case caseRecv:
			sg = c.sendq.dequeue() // 当通道有等待的sender
			if sg != nil {
				goto recv
			}
			if c.qcount > 0 { // 当前通道有缓存数据，则从通道里面读取
				goto bufrecv
			}
			if c.closed != 0 { // 当前通道已关闭
				goto rclose
			}

		case caseSend:
			if raceenabled {
				racereadpc(c.raceaddr(), cas.pc, chansendpc)
			}
			if c.closed != 0 { // 当前通道已关闭，直接panic
				goto sclose
			}
			sg = c.recvq.dequeue() // 当通道有等待的recevier
			if sg != nil {
				goto send
			}
			if c.qcount < c.dataqsiz { // 当前buffered通道还有可用缓存空间
				goto bufsend
			}

		case caseDefault: // default分支
			dfli = casi
			dfl = cas
		}
	}

	if dfl != nil { // 所有case分支都未就绪，那就执行default分支
		selunlock(scases, lockorder) // 将所有scase上的通道unlock
		casi = dfli
		cas = dfl
		goto retc
	}

	// pass 2 - enqueue on all chans
	gp = getg()
	if gp.waiting != nil { // 当前g处理waiting状态，非运行状态，这是异常情况
		throw("gp.waiting != nil")
	}
	nextp = &gp.waiting // 此gp.waiting == nil
	for _, casei := range lockorder {
		casi = int(casei)
		cas = &scases[casi]
		if cas.kind == caseNil {
			continue
		}
		c = cas.c
		sg := acquireSudog() // 从二级缓存中获取sudog结构，尽量复用，避免对象分配
		sg.g = gp
		sg.isSelect = true // 标志当g处在select语句中
		// No stack splits between assigning elem and enqueuing
		// sg on gp.waiting where copystack can find it.
		sg.elem = cas.elem
		sg.releasetime = 0
		if t0 != 0 {
			sg.releasetime = -1
		}
		sg.c = c
		// Construct waiting list in lock order.
		*nextp = sg // 将sg挂到gp.waiting上
		nextp = &sg.waitlink

		switch cas.kind {
		case caseRecv:
			c.recvq.enqueue(sg) // sudog入receiver队列

		case caseSend:
			c.sendq.enqueue(sg) // sudog入sender队列
		}
	}

	// wait for someone to wake us up
	gp.param = nil // 当g从waiting状态唤醒后，param指向唤醒的对象sudog信息，这里面先清空，等待被唤醒
	// Signal to anyone trying to shrink our stack that we're about
	// to park on a channel. The window between when this G's status
	// changes and when we set gp.activeStackChans is not safe for
	// stack shrinking.
	atomic.Store8(&gp.parkingOnChan, 1)
	gopark(selparkcommit, nil, waitReasonSelect, traceEvGoBlockSelect, 1) // 挂起g

	// 由于select中某个case的通道可读或者可写，当前G被唤醒，并被调度执行后开始执行下面代码
	gp.activeStackChans = false

	sellock(scases, lockorder) // 将所有通道都lock住

	gp.selectDone = 0       // 标志当前select是否完成
	sg = (*sudog)(gp.param) // 获取唤醒G的sudog信息，sudog里面包含对应的通道信息
	gp.param = nil

	// pass 3 - dequeue from unsuccessful chans
	// otherwise they stack up on quiet channels
	// record the successful case, if any.
	// We singly-linked up the SudoGs in lock order.
	casi = -1
	cas = nil
	sglist = gp.waiting
	// Clear all elem before unlinking from gp.waiting.
	for sg1 := gp.waiting; sg1 != nil; sg1 = sg1.waitlink {
		// 清空gp.waiting链表上sudog上的通道相关信息
		// 存不存在清除了不能清空的通道信息？不存在。gp.waiting存在两个情况：
		/*
			情况1：
				go func() {
					ch <- 1 // waiting链表只有一个元素，清空没问题
				}()

			情况2:
			 select {
			 case <-ch:
			 case <-ch2:
			 case ch3<-1:
			 }
			 情况2清空也没问题，因为通道选择器一次最多只选择一个通道，如果选择ch，清空sudog中记录的ch2/ch3也是没问题的
		*/
		sg1.isSelect = false
		sg1.elem = nil
		sg1.c = nil
	}
	gp.waiting = nil

	for _, casei := range lockorder {
		k = &scases[casei]
		if k.kind == caseNil {
			continue
		}
		if sglist.releasetime > 0 {
			k.releasetime = sglist.releasetime
		}
		if sg == sglist { // 唤醒G的sudog对象，在挂载到g.param上已从sendq或recvq出队了
			// sg has already been dequeued by the G that woke us up.
			casi = int(casei)
			cas = k
		} else {
			c = k.c
			if k.kind == caseSend {
				c.sendq.dequeueSudoG(sglist) // 从sender队列中drop掉sglist
			} else {
				c.recvq.dequeueSudoG(sglist) // 从receiver队列中drop掉sglist
			}
		}
		sgnext = sglist.waitlink
		sglist.waitlink = nil
		releaseSudog(sglist)
		sglist = sgnext
	}

	if cas == nil {
		// We can wake up with gp.param == nil (so cas == nil)
		// when a channel involved in the select has been closed.
		// It is easiest to loop and re-run the operation;
		// we'll see that it's now closed.
		// Maybe some day we can signal the close explicitly,
		// but we'd have to distinguish close-on-reader from close-on-writer.
		// It's easiest not to duplicate the code and just recheck above.
		// We know that something closed, and things never un-close,
		// so we won't block again.

		// 当select是由于通道被关闭唤醒，那么case为nil，那么继续执行loop逻辑
		goto loop
	}

	c = cas.c

	if debugSelect {
		print("wait-return: cas0=", cas0, " c=", c, " cas=", cas, " kind=", cas.kind, "\n")
	}

	if cas.kind == caseRecv {
		recvOK = true
	}

	if raceenabled {
		if cas.kind == caseRecv && cas.elem != nil {
			raceWriteObjectPC(c.elemtype, cas.elem, cas.pc, chanrecvpc)
		} else if cas.kind == caseSend {
			raceReadObjectPC(c.elemtype, cas.elem, cas.pc, chansendpc)
		}
	}
	if msanenabled {
		if cas.kind == caseRecv && cas.elem != nil {
			msanwrite(cas.elem, c.elemtype.size)
		} else if cas.kind == caseSend {
			msanread(cas.elem, c.elemtype.size)
		}
	}

	selunlock(scases, lockorder)
	goto retc

bufrecv:
	// can receive from buffer
	if raceenabled {
		if cas.elem != nil {
			raceWriteObjectPC(c.elemtype, cas.elem, cas.pc, chanrecvpc)
		}
		raceacquire(chanbuf(c, c.recvx))
		racerelease(chanbuf(c, c.recvx))
	}
	if msanenabled && cas.elem != nil {
		msanwrite(cas.elem, c.elemtype.size)
	}
	recvOK = true
	qp = chanbuf(c, c.recvx)
	if cas.elem != nil {
		typedmemmove(c.elemtype, cas.elem, qp)
	}
	typedmemclr(c.elemtype, qp)
	c.recvx++
	if c.recvx == c.dataqsiz {
		c.recvx = 0
	}
	c.qcount--
	selunlock(scases, lockorder)
	goto retc

bufsend:
	// can send to buffer
	if raceenabled {
		raceacquire(chanbuf(c, c.sendx))
		racerelease(chanbuf(c, c.sendx))
		raceReadObjectPC(c.elemtype, cas.elem, cas.pc, chansendpc)
	}
	if msanenabled {
		msanread(cas.elem, c.elemtype.size)
	}
	typedmemmove(c.elemtype, chanbuf(c, c.sendx), cas.elem)
	c.sendx++
	if c.sendx == c.dataqsiz {
		c.sendx = 0
	}
	c.qcount++
	selunlock(scases, lockorder)
	goto retc

recv:
	// can receive from sleeping sender (sg)
	/* recv处理逻辑：
	1. 当前通道是unbuffered 通道，则直接将sender数据handoff给当前接收者
	2. 若是buffered通道，则从通道缓存中读取数据给当前接收者

	无论1和2最后都会waiting状态的sender切换runable状态，等待再次调度
	*/
	recv(c, sg, cas.elem, func() { selunlock(scases, lockorder) }, 2)
	if debugSelect {
		print("syncrecv: cas0=", cas0, " c=", c, "\n")
	}
	recvOK = true
	goto retc

rclose:
	// read at end of closed channel
	selunlock(scases, lockorder)
	recvOK = false
	if cas.elem != nil {
		typedmemclr(c.elemtype, cas.elem)
	}
	if raceenabled {
		raceacquire(c.raceaddr())
	}
	goto retc

send:
	// can send to a sleeping receiver (sg)
	if raceenabled {
		raceReadObjectPC(c.elemtype, cas.elem, cas.pc, chansendpc)
	}
	if msanenabled {
		msanread(cas.elem, c.elemtype.size)
	}
	send(c, sg, cas.elem, func() { selunlock(scases, lockorder) }, 2)
	if debugSelect {
		print("syncsend: cas0=", cas0, " c=", c, "\n")
	}
	goto retc

retc:
	if cas.releasetime > 0 {
		blockevent(cas.releasetime-t0, 1)
	}
	return casi, recvOK

sclose:
	// send on closed channel
	selunlock(scases, lockorder)
	panic(plainError("send on closed channel"))
}

func (c *hchan) sortkey() uintptr {
	return uintptr(unsafe.Pointer(c))
}

// A runtimeSelect is a single case passed to rselect.
// This must match ../reflect/value.go:/runtimeSelect
type runtimeSelect struct {
	dir selectDir
	typ unsafe.Pointer // channel type (not used here)
	ch  *hchan         // channel
	val unsafe.Pointer // ptr to data (SendDir) or ptr to receive buffer (RecvDir)
}

// These values must match ../reflect/value.go:/SelectDir.
type selectDir int

const (
	_             selectDir = iota
	selectSend              // case Chan <- Send
	selectRecv              // case <-Chan:
	selectDefault           // default
)

//go:linkname reflect_rselect reflect.rselect
func reflect_rselect(cases []runtimeSelect) (int, bool) {
	if len(cases) == 0 {
		block()
	}
	sel := make([]scase, len(cases))
	order := make([]uint16, 2*len(cases))
	for i := range cases {
		rc := &cases[i]
		switch rc.dir {
		case selectDefault:
			sel[i] = scase{kind: caseDefault}
		case selectSend:
			sel[i] = scase{kind: caseSend, c: rc.ch, elem: rc.val}
		case selectRecv:
			sel[i] = scase{kind: caseRecv, c: rc.ch, elem: rc.val}
		}
		if raceenabled || msanenabled {
			selectsetpc(&sel[i])
		}
	}

	return selectgo(&sel[0], &order[0], len(cases))
}

func (q *waitq) dequeueSudoG(sgp *sudog) {
	x := sgp.prev
	y := sgp.next
	if x != nil {
		if y != nil {
			// middle of queue
			x.next = y
			y.prev = x
			sgp.next = nil
			sgp.prev = nil
			return
		}
		// end of queue
		x.next = nil
		q.last = x
		sgp.prev = nil
		return
	}
	if y != nil {
		// start of queue
		y.prev = nil
		q.first = y
		sgp.next = nil
		return
	}

	// x==y==nil. Either sgp is the only element in the queue,
	// or it has already been removed. Use q.first to disambiguate.
	if q.first == sgp {
		q.first = nil
		q.last = nil
	}
}
