## pool概述

> A Pool is a set of temporary objects that may be individually saved and retrieved.

> Any item stored in the Pool may be removed automatically at any time without notification. If the Pool holds the only reference when this happens, the item might be deallocated.

> A Pool is safe for use by multiple goroutines simultaneously.

> Pool's purpose is to cache allocated but unused items for later reuse, relieving pressure on the garbage collector. That is, it makes it easy to build efficient, thread-safe free lists. However, it is not suitable for all free lists

sync.Pool提供了临时对象缓存池，存在池子的对象可能在任何时刻被自动移除，我们对此不能做任何预期。sync.Pool**可以并发使用**，它通过**复用对象来减少对象内存分配和GC的压力**。当负载大的时候，临时对象缓存池会扩大，**缓存池中的对象会在每2个GC循环中清除**。

sync.Pool拥有两个对象存储容器：`local pool`和`victim cache`，当获取对象时，优先从`victvim cache`中检索，若未找到则再从`local pool`中检索，若也未获取到，则调用New方法创建一个对象返回。当对象放回sync.Pool时候，会放在`local pool`中。当GC开始时候，首选将`victim cache`中所有对象清除，然后将`local pool`容器中所有对象都会移动到`victim cache`中，所以说缓存池中的对象会在每2个GC循环中清除。

## 用法

sync.Pool提供两个接口，`Get`和`Put`分别用于从缓存池中获取临时对象，和将临时对象放回到缓存池中：

```go
func (p *Pool) Get() interface{}
func (p *Pool) Put(x interface{})
```

### 示例1

```go

type A struct {
	Name string
}

func (a *A) Reset() {
	a.Name = ""
}

var pool = sync.Pool{
	New: func() interface{} {
		return new(A)
	},
}

func main() {
	objA := pool.Get().(*A)
	objA.Reset() // 重置一下对象数据，防止脏数据
	defer pool.Put(objA)
	objA.Name = "test123"
	fmt.Println(objA)
}
```

接下来我们进行基准测试下未使用和使用sync.Pool情况：

```go
type A struct {
	Name string
}

func (a *A) Reset() {
	a.Name = ""
}

var pool = sync.Pool{
	New: func() interface{} {
		return new(A)
	},
}

func BenchmarkWithoutPool(b *testing.B) {
	var a *A
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			a = new(A)
			a.Name = "tink"
		}
	}
}

func BenchmarkWithPool(b *testing.B) {
	var a *A
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			a = pool.Get().(*A)
			a.Reset()
			a.Name = "tink"
			pool.Put(a) // 一定要记得放回操作，否则退化到每次都需要New操作
		}
	}
}
```

基准测试结果如下：

```
# go test -benchmem -run=^$ -bench  .
goos: darwin
goarch: amd64
BenchmarkWithoutPool-8              3404            314232 ns/op          160001 B/op      10000 allocs/op
BenchmarkWithPool-8                 5870            220399 ns/op               0 B/op          0 allocs/op
```
从上面基准测试中，我们可以看到使用sync.Pool之后，每次执行的耗时由314232ns降到220399ns，降低了29.8%，每次执行的内存分配降到0（注意这是平均值，并不是没进行过内存分配，只不过是绝大数操作没有进行过内存分配，最终平均下来远小于1，四舍五入为0）。

### 示例2

[go-redis/redis]项目中实现连接池时候，使用到sync.Pool来创建定时器：

```go
// 创建timer Pool
var timers = sync.Pool{
	New: func() interface{} { // 定义创建临时对象创建方法
		t := time.NewTimer(time.Hour)
		t.Stop()
		return t
	},
}

func (p *ConnPool) waitTurn(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	...
	timer := timers.Get().(*time.Timer) // 从缓存池中取出对象
	timer.Reset(p.opt.PoolTimeout)

	select {
	...
	case <-timer.C:
		timers.Put(timer) // 将对象放回到缓存池中，以便下次使用
		atomic.AddUint32(&p.stats.Timeouts, 1)
		return ErrPoolTimeout
	}
```

## 数据结构

![](https://static.cyub.vip/images/202106/pool.png)

sync.Pool底层数据结构体是Pool结构体([sync/pool.go](https://github.com/golang/go/blob/go1.14.13/src/sync/pool.go#L44-L57))：

```go
type Pool struct {
	noCopy noCopy // nocopy机制，用于go vet命令检查是否复制后使用

	local     unsafe.Pointer // 指向[P]poolLocal数组，P等于runtime.GOMAXPROCS(0)
	localSize uintptr        // local数组大小，即[P]poolLocal大小

	victim     unsafe.Pointer // 指向上一个gc循环前的local
	victimSize uintptr        // victim数组大小

	New func() interface{} // 创建临时对象的方法，当从local数组和victim数组中都没有找到临时对象缓存，那么会调用此方法现场创建一个
}
```

Pool.local指向大小为`runtime.GOMAXPROCS(0)`的poolLocal数组，相当于大小为``runtime.GOMAXPROCS(0)`的缓存槽(solt)。每一个P都会通过其ID关联一个槽位上的poolLocal，比如对于ID=1的P关联的poolLocal就是[1]poolLocal，这个poolLocal属于per-P级别的poolLocal，与P关联的M和G可以无锁的操作此poolLocal。

poolLocal结构如下：

```go
type poolLocal struct {
	poolLocalInternal // 内嵌poolLocalInternal结构体
	// 进行一些padding，阻止false share
	pad [128 - unsafe.Sizeof(poolLocalInternal{})%128]byte
}

type poolLocalInternal struct {
	private interface{} // 私有属性，快速存取临时对象
	shared  poolChain   // shared是一个双端链表
}
```

为啥不直接把所有poolLocalInternal字段都写到poolLocal里面，而是采用内嵌形式？这是为了好计算出poolLocal的padding大小。

poolChain结构如下：

```go
type poolChain struct {
	// 指向双向链表头
	head *poolChainElt

	// 指向双向链表尾
	tail *poolChainElt
}

type poolChainElt struct {
	poolDequeue
	next, prev *poolChainElt
}

type poolDequeue struct {
	// headTail高32位是环形队列的head
	// headTail低32位是环形队列的tail
	// [tail, head)范围是队列所有元素
	headTail uint64

	vals []eface // 用于存放临时对象，大小是2的倍数，最小尺寸是8，最大尺寸是dequeueLimit
}

type eface struct {
	typ, val unsafe.Pointer
}
```

`poolLocalInternal`的shared字段指向是一个双向链表(doubly-linked list)，链表每一个元素都是poolChainElt类型，poolChainElt是一个双端队列（Double-ended Queue简写deque），并且链表中每一个元素的队列大小是2的倍数，且是前一个元素队列大小的2倍。poolChainElt是基于环形队列(circular queue)实现的双端队列。

若poolLocal属于当前P，那么可以对shared进行pushHead和popHead操作，而其他P只能进行popTail操作。当前其他P进行popTail操作时候，会检查链表中节点的poolChainElt是否为空，若是空，则会drop掉该节点，这样当popHead操作时候避免去查一个空的poolChainElt。

`poolDequeue`中的headTail字段的高32位记录的是环形队列的head，其低32位是环形队列的tail。vals是环形队列的底层数组。

## Get操作

我们来看下如何从sync.Pool中取出临时对象。下面代码已去掉竞态检测相关代码。

```go
func (p *Pool) Get() interface{} {
	l, pid := p.pin() // 返回当前per-P级poolLocal和P的id
	x := l.private
	l.private = nil
	if x == nil {
		x, _ = l.shared.popHead()
		if x == nil {
			x = p.getSlow(pid)
		}
	}
	runtime_procUnpin()
	if x == nil && p.New != nil {
		x = p.New()
	}
	return x
}
```

上面代码执行流程如下：

1. 首先通过调用pin方法，获取当前G关联的P对应的poolLocal和该P的id
2. 接着查看poolLocal的private字段是否存放了对象，如果有的话，那么该字段存放的对象就直接返回，这属于最快路径。
3. 若poolLocal的private字段未存放对象，那么就尝试从poolLocal的双端队列中取出对象，这个操作是lock-free的。
4. 若G关联的per-P级poolLocal的双端队列中没有取出来，那么就尝试从其他P关联的poolLocal中偷一个。若从其他P关联的poolLocal没有偷到一个，那么就尝试从victim中取。
5. 若步骤4中没有取到缓存对象，那么只能调用pool.New方法新创建一个对象。

我们来看下pin方法：

```go
func (p *Pool) pin() (*poolLocal, int) {
	pid := runtime_procPin() // 禁止M被抢占
	s := atomic.LoadUintptr(&p.localSize) // 原子性加载local pool的大小
	l := p.local
	if uintptr(pid) < s {
		// 如果local pool大小大于P的id，那么从local pool取出来P关联的poolLocal
		return indexLocal(l, pid), pid
	}

	/*
	 * 当p.local指向[P]poolLocal数组还没有创建
	 * 或者通过runtime.GOMAXPROCS()调大P数量时候都可能会走到此处逻辑
	 */
	return p.pinSlow()
}

func (p *Pool) pinSlow() (*poolLocal, int) {
	runtime_procUnpin()
	allPoolsMu.Lock() // 加锁
	defer allPoolsMu.Unlock()
	pid := runtime_procPin()

	s := p.localSize
	l := p.local
	if uintptr(pid) < s { // 加锁后再次判断一下P关联的poolLocal是否存在
		return indexLocal(l, pid), pid
	}
	if p.local == nil { // 将p记录到全局变量allPools中，执行GC钩子时候，会使用到
		allPools = append(allPools, p)
	}

	size := runtime.GOMAXPROCS(0) // 根据P数量创建p.local
	local := make([]poolLocal, size)
	atomic.StorePointer(&p.local, unsafe.Pointer(&local[0]))
	atomic.StoreUintptr(&p.localSize, uintptr(size))
	return &local[pid], pid
}

func indexLocal(l unsafe.Pointer, i int) *poolLocal {
	// 通过uintptr和unsafe.Pointer取出[P]poolLocal数组中，索引i对应的poolLocal
	lp := unsafe.Pointer(uintptr(l) + uintptr(i)*unsafe.Sizeof(poolLocal{}))
	return (*poolLocal)(lp)
}
```

pin方法中会首先调用`runtime_procPin`来设置M禁止被抢占。GMP调度模型中，M必须绑定到P之后才能执行G，禁止M被抢占就是禁止M绑定的P被剥夺走，相当于`pin processor`。

pin方法中为啥要首先禁止M被抢占？这是因为我们需要找到per-P级的poolLocal，如果在此过程中发生M绑定的P被剥夺，那么我们找到的就可能是其他M的per-P级poolLocal。

`runtime_procPin`方法是通过给M加锁实现禁止被抢占，即`m.locks++`，当`m.locks==0`时候m是可以被抢占的:

```go
//go:linkname sync_runtime_procPin sync.runtime_procPin
//go:nosplit
func sync_runtime_procPin() int {
	return procPin()
}

//go:linkname sync_runtime_procUnpin sync.runtime_procUnpin
//go:nosplit
func sync_runtime_procUnpin() {
	procUnpin()
}

//go:nosplit
func procPin() int {
	_g_ := getg()
	mp := _g_.m

	mp.locks++ // 给m加锁
	return int(mp.p.ptr().id)
}

//go:nosplit
func procUnpin() {
	_g_ := getg()
	_g_.m.locks--
}
```

`go:linkname`是编译指令用于将私有函数或者变量在编译阶段链接到指定位置。从上面代码中我们可以看到`sync.runtime_procPin`和`sync.runtime_procUnpin`最终实现方法是`sync_runtime_procPin`和`sync_runtime_procUnpin`。

pinSlow方法用到的`allPoolsMu`和`allPools`是全局变量：

```go
var (
	allPoolsMu Mutex

	// allPools is the set of pools that have non-empty primary
	// caches. Protected by either 1) allPoolsMu and pinning or 2)
	// STW.
	allPools []*Pool

	// oldPools is the set of pools that may have non-empty victim
	// caches. Protected by STW.
	oldPools []*Pool
)
```

接下我们来看Get流程中步骤3的实现：

```go
func (c *poolChain) popHead() (interface{}, bool) {
	d := c.head // 从双向链表的头部开始
	for d != nil {
		if val, ok := d.popHead(); ok { // 从双端队列头部取对象缓存，若取到则返回
			return val, ok
		}
		// 若未取到，则尝试从上一个节点开始取
		d = loadPoolChainElt(&d.prev) 
	}
	return nil, false
}

func loadPoolChainElt(pp **poolChainElt) *poolChainElt {
	return (*poolChainElt)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(pp))))
}
```

最后我们看下Get流程中步骤4的实现：

```go
func (p *Pool) getSlow(pid int) interface{} {
	size := atomic.LoadUintptr(&p.localSize)
	locals := p.local
	for i := 0; i < int(size); i++ {
		// 尝试从其他P关联的poolLocal取一个，
		// 类似GMP调度模型从其他P的runable G队列中偷一个

		// 偷的时候是双向链表尾部开始偷，这个和从本地P的poolLocal取恰好是反向的
		l := indexLocal(locals, (pid+i+1)%int(size))
		if x, _ := l.shared.popTail(); x != nil {
			return x
		}
	}

	// 若从其他P的poolLocal没有偷到，则尝试从victim cache取
	size = atomic.LoadUintptr(&p.victimSize)
	if uintptr(pid) >= size {
		return nil
	}
	locals = p.victim
	l := indexLocal(locals, pid)
	if x := l.private; x != nil {
		l.private = nil
		return x
	}
	for i := 0; i < int(size); i++ {
		l := indexLocal(locals, (pid+i)%int(size))
		if x, _ := l.shared.popTail(); x != nil {
			return x
		}
	}

	atomic.StoreUintptr(&p.victimSize, 0)

	return nil
}

func (c *poolChain) popTail() (interface{}, bool) {
	d := loadPoolChainElt(&c.tail)
	if d == nil {
		return nil, false
	}

	for {
		d2 := loadPoolChainElt(&d.next)

		if val, ok := d.popTail(); ok { // 从双端队列的尾部出队
			return val, ok
		}

		if d2 == nil { // 若下一个节点为空，则返回。说明链表已经遍历完了
			return nil, false
		}

		// 下面代码会将当前节点的上一个节点从链表中删除掉。
		// 为什么要删掉它，因为该节点的队列里面有没有对象缓存了，
		// 删掉之后，下次本地P取的时候，不必遍历此空节点了
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&c.tail)), unsafe.Pointer(d), unsafe.Pointer(d2)) {
			storePoolChainElt(&d2.prev, nil)
		}
		d = d2
	}
}

func storePoolChainElt(pp **poolChainElt, v *poolChainElt) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(pp)), unsafe.Pointer(v))
}
```

我们画出Get流程中步骤3和4的中从`local pool`取对象示意图：

![](https://static.cyub.vip/images/202106/pool_queue_pop.png)



## 进一步阅读

- [Go: Understand the Design of Sync.Pool](https://medium.com/a-journey-with-go/go-understand-the-design-of-sync-pool-2dde3024e277)