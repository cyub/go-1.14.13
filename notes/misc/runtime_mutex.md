# runtime.mutex

Go内部有两个互斥锁，一个用户态的，暴露出来给我们使用，一个是runtime的。用户态的mutex，即sync.Mutex，它使用semaphore实现的，最终会导致未抢到锁的goroutine休眠，锁的级别是G，即会发生goroutine switches，它的时延是ns级别。runtime态的mutex使用的futex，锁的级别是M，即会发生thread switches，它的时延是μs级别的。下面是两种mutex锁GMP情况。

锁类型 | G | M | P |
--- | --- | --- | ---
runtime.mutex | Y | Y |Y
sync.Mutex | Y | N | N

runtime态mutex使用futex实现。当没有竞争时，futex操作不会陷入内核，直接拿到锁，否则futex陷入内核，进行sleep操作，线程上下文发生切换。需要注意的是go采用的是混合型的futex，陷入内核之前会先自旋spin一会。futex系统调用使用了一个hash表来管理等待的 thread，这个hash表每个bucket也需要一个 lock，内核使用spinlock来实现。

sync.Mutex使用semaphore实现。信号变量地址组成一个平衡树，信号变量地址会通过hash桶算法定位到哪一个bucket上面的平衡树，对于每一个bucket都需要一个锁，这个锁就是使用runtime.mutex实现的。

![Go semaphore 和futex 使用lock分析](https://static.cyub.vip/images/202107/futex-spin.png)

对于sync.Mutex，其与runtime.mutex关系如下：

![](https://static.cyub.vip/images/202107/go-futex.png)

## 源码分析

runtime.mutex实现上，在linux系统上使用futex，对于window系统使用semaphore实现的（注意这里面是系统调用，与sync.Mutex的seamphore不一样）。这里面分析的是基于futex实现的。

```go
// Mutual exclusion locks.  In the uncontended case,
// as fast as spin locks (just a few user-level instructions),
// but on the contention path they sleep in the kernel.
// A zeroed Mutex is unlocked (no need to initialize each lock).
type mutex struct {
	// Futex-based impl treats it as uint32 key,
	// while sema-based impl as M* waitm.
	// Used to be a union, but unions break precise GC.
	key uintptr
}
```

runtime.mutex提供了lock和unlock接口用来加锁和释放锁操作。

```go

// This implementation depends on OS-specific implementations of
//
//	futexsleep(addr *uint32, val uint32, ns int64)
//		Atomically,
//			if *addr == val { sleep }
//		Might be woken up spuriously; that's allowed.
//		Don't sleep longer than ns; ns < 0 means forever.
//
//	futexwakeup(addr *uint32, cnt uint32)
//		If any procs are sleeping on addr, wake up at most cnt.

const (
	mutex_unlocked = 0
	mutex_locked   = 1
	mutex_sleeping = 2

	active_spin     = 4
	active_spin_cnt = 30
	passive_spin    = 1
)

// Possible lock states are mutex_unlocked, mutex_locked and mutex_sleeping.
// mutex_sleeping means that there is presumably at least one sleeping thread.
// Note that there can be spinning threads during all states - they do not
// affect mutex's state.

// We use the uintptr mutex.key and note.key as a uint32.
//go:nosplit
func key32(p *uintptr) *uint32 {
	return (*uint32)(unsafe.Pointer(p))
}

func lock(l *mutex) {
	gp := getg()

	if gp.m.locks < 0 {
		throw("runtime·lock: lock count")
	}
	gp.m.locks++

	// Speculative grab for lock.
	v := atomic.Xchg(key32(&l.key), mutex_locked)
	if v == mutex_unlocked {
		return
	}

	// wait is either MUTEX_LOCKED or MUTEX_SLEEPING
	// depending on whether there is a thread sleeping
	// on this mutex. If we ever change l->key from
	// MUTEX_SLEEPING to some other value, we must be
	// careful to change it back to MUTEX_SLEEPING before
	// returning, to ensure that the sleeping thread gets
	// its wakeup call.
	wait := v

	// On uniprocessors, no point spinning.
	// On multiprocessors, spin for ACTIVE_SPIN attempts.
	spin := 0
	if ncpu > 1 { // 多核cpu才进行自旋，否则没意义
		spin = active_spin
	}
	for {
		// Try for lock, spinning.
		for i := 0; i < spin; i++ {
			for l.key == mutex_unlocked {
				if atomic.Cas(key32(&l.key), mutex_unlocked, wait) {
					return
				}
			}
			procyield(active_spin_cnt) // procyield是汇编语言实现。函数内部循环调用PAUSE指令。PAUSE指令什么都不做，但是会消耗CPU时间
		}

		// Try for lock, rescheduling.
		for i := 0; i < passive_spin; i++ {
			for l.key == mutex_unlocked {
				if atomic.Cas(key32(&l.key), mutex_unlocked, wait) {
					return
				}
			}
			osyield() // 执行sched_yield系统调用，主动让出cpu时间片
		}

		// Sleep.
		v = atomic.Xchg(key32(&l.key), mutex_sleeping)
		if v == mutex_unlocked {
			return
		}
		wait = mutex_sleeping
		futexsleep(key32(&l.key), mutex_sleeping, -1) // futex系统调用
	}
}

func unlock(l *mutex) {
	v := atomic.Xchg(key32(&l.key), mutex_unlocked)
	if v == mutex_unlocked {
		throw("unlock of unlocked lock")
	}
	if v == mutex_sleeping {
		futexwakeup(key32(&l.key), 1)
	}

	gp := getg()
	gp.m.locks--
	if gp.m.locks < 0 {
		throw("runtime·unlock: lock count")
	}
	if gp.m.locks == 0 && gp.preempt { // restore the preemption request in case we've cleared it in newstack
		gp.stackguard0 = stackPreempt
	}
}
```

## 资料

- [let's talk locks](https://speakerdeck.com/kavya719/lets-talk-locks)
- [Linux Lock 与 Golang Mutex 的实现与性能分析](https://blog.xiaokezhao.com/locks-golang-mutex-implementation-performance-measure/)