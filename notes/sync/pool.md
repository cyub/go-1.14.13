## pool概述

> A Pool is a set of temporary objects that may be individually saved and retrieved.

> Any item stored in the Pool may be removed automatically at any time without notification. If the Pool holds the only reference when this happens, the item might be deallocated.

> A Pool is safe for use by multiple goroutines simultaneously.

> Pool's purpose is to cache allocated but unused items for later reuse, relieving pressure on the garbage collector. That is, it makes it easy to build efficient, thread-safe free lists. However, it is not suitable for all free lists

sync.Pool提供了临时对象缓存池，存在池子的对象可能在任何时刻被自动移除，我们对此不能做任何预期。sync.Pool可以并发使用，它通过**复用对象来减少对象内存分配和GC的压力**。当负载大的时候，临时对象缓存池会扩大，**缓存池中的对象会在每2个GC循环中清除**。

sync.Pool拥有两个对象存储容器：`local pool`和`victim cache`，当获取对象时，优先从`victvim cache`中检索，若未找到则再从`local pool`中检索，若也未获取到，则调用New方法创建一个对象返回。当对象放回sync.Pool时候，会放在`local pool`中。当GC开始时候，首选将`victim cache`中所有对象清除，然后将`local pool`容器中所有对象都会移动到`victim cache`中，所以说缓存池中的对象会在每2个GC循环中清除。

## 用法

sync.Pool提供了Get()和Put()方法用于从缓存池中获取临时对象，和将临时对象放回到缓冲池中：

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
从上面基准测试中，我们可以看到使用sync.Pool之后，执行耗时降低了29.8%。

## 数据结构

![](https://static.cyub.vip/images/202106/pool.png)

sync.Pool底层数据结构体是Pool结构体([sync/pool.go](https://github.com/golang/go/blob/go1.14.13/src/sync/pool.go#L44-L57))：

```go
type Pool struct {
	noCopy noCopy // nocopy机制，用于go vet命令检查是否复制后使用

	local     unsafe.Pointer // 指向[P]poolLocal数组，P数量等于runtime.GOMAXPROCS(0)
	localSize uintptr        // local数组大小，即[P]poolLocal大小

	victim     unsafe.Pointer // 指向上一个gc循环前的local
	victimSize uintptr        // victims数组大小

	New func() interface{} // 创建临时对象的方法，当从local数组和victim数组没有找到临时对象缓存，那么会调用此方法现场创建一个
}
```

Pool.local指向大小为`runtime.GOMAXPROCS(0)`的poolLocal数组。poolLocal结构如下：

```go
type poolLocal struct {
	poolLocalInternal // 内嵌poolLocalInternal结构体，
	// 为啥不直接把所有poolLocalInternal字段写到poolLocal里面，这是为了好计算出pad大小

	// 进行一些padding，阻止false share
	pad [128 - unsafe.Sizeof(poolLocalInternal{})%128]byte
}

type poolLocalInternal struct {
	private interface{} // 私有属性，快速存取临时对象
	shared  poolChain   // shared是一个双端链表，本地P能够pushHead/popHead，任意P都能够popTail.
}
```

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
	// headTail高32位是循环队列的head
	// headTail低32位是循环队列的tail
	// [tail, head)范围是队列所有元素
	headTail uint64

	vals []eface // 用于存放临时对象，大小是2的倍数，最小大小是8
}

type eface struct {
	typ, val unsafe.Pointer
}
```

`poolLocalInternal`的shared字段指向是一个双向链表(doubly-linked list)，链表每一个元素都是poolChainElt类型，poolChainElt是一个双端队列（Double-ended Queue简写deque），并且链表中每一个元素的队列大小是2的倍数，且是前一个元素队列大小的2倍。poolChainElt是基于环形队列(circular queue)实现的双端队列。
