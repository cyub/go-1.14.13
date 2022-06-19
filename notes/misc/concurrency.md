# 并发相关术语

## CAS

CAS是Compare and Swap的缩写(或者Check and Swap)。CAS是一种无锁算法，它通过处理器的指令(CMPXCHG指令)来保证操作的原子性，它包含三个操作数：

- 变量内存地址，V表示
- 旧的预期值，A表示
- 准备设置的新值，B表示

当执行CAS指令时，只有当V等于A时，才会用B去更新V的值，否则就不会执行更新操作。

CAS使用可以避免使用系统锁的开销，但它不是银弹，它主要存在以下3个问题：
- ABA问题

    由于CAS只关注最开始状态和最终的状态，无法处理中间状态，这有可能会造成信息丢失的问题。

- 自旋时间不可预期，可能会导致开销大

    在使用CAS实现自旋锁时候，没法知道自旋啥时候结束，这会CPU开销过大，有可能一直等待中。也就是说CAS实现了lock-free，但没法保证wait-free。

- 只能保证1个共享变量的原子操作

CAS常见应用是实现自旋锁(spin lock)，以及实现无锁数据结构，比如无锁队列，无锁栈。

## ABA

ABA问题是使用CAS原语实现无锁数据结构中存在的问题。ABA问题的描述如下：

假设两个线程T1和T2访问同一个变量V，当T1访问变量V时，读取到V的值为A；此时线程T1被抢占了，T2开始执行，T2先将变量V的值从A变成了B，然后又将变量V从B变回了A；此时T1又抢占了主动权，继续执行，它发现变量V的值还是A，以为没有发生变化，所以就继续执行了。这个过程中，变量V从A变为B，再由B变为A就被形象地称为ABA问题了。

ABA案例：使用CAS+链表实现的无锁堆栈。

假定无锁堆栈的栈顶为A，这时线程T1已经知道A.next为B（A-->B)：

```java
Node* currentNode = this->head; // A
Node* nextNode = currentNode->next; // B
```

然后希望将栈顶A弹出去，只需使用CAS将A替换成B：

```java
this->head.compareAndSet(currentNode, nextNode);
```

在T1执行上面这条指令之前，线程T2介入，将A、B出栈，再依次push D、C、A，此时堆栈结构为(A-->C-->D)，而对象B此时处于游离状态。

此时轮到线程T1执行CAS操作，检测发现栈顶仍为A，所以CAS成功，栈顶变为B，但实际上B.next为null，所以此时的情况变为堆栈中只有B一个元素，C和D组成的链表不再存在于堆栈中，平白无故就把C、D丢掉了。


### 解决方案

解决ABA办法可以采用[Hazard pointer](https://en.wikipedia.org/wiki/Hazard_pointer)。

Java中提供了AtomicStampedReference和AtomicMarkableReference来解决ABA问题，它在原有类的基础上，除了比较与修改期待的值外，增加了一个时间戳。对时间戳也进行CAS操作。这也称为双重CAS。每次修改一个结点，其时间戳都发生变化。这样即使共享一个复用结点，最终CAS也能返回正常的结果。


## false sharing

false sharing中文翻译为伪共享。

什么是伪共享呢？在计算机组成中，CPU 的运算速度比内存高出几个数量级，为了 CPU 能够更高效地与内存进行交互，在 CPU 和内存之间设计了多层缓存机制，如下图所示：

![](https://static.cyub.vip/images/202206/cpu-cache.png)

一般来说，CPU 会分为三级缓存，分别为L1 一级缓存、L2 二级缓存和L3 三级缓存。越靠近 CPU 的缓存，速度越快，但是缓存的容量也越小。所以从性能上来说，L1 > L2 > L3，容量方面 L1 < L2 < L3。CPU 读取数据时，首先会从 L1 查找，如果未命中则继续查找 L2，如果还未能命中则继续查找 L3，最后还没命中的话只能从内存中查找，读取完成后再将数据逐级放入缓存中。此外，多线程之间共享一份数据的时候，需要其中一个线程将数据写回主存，其他线程访问主存数据。

由此可见，引入多级缓存是为了能够让 CPU 利用率最大化。如果你在做频繁的 CPU 运算时，需要尽可能将数据保持在缓存中。那么 CPU 从内存中加载数据的时候，是如何提高缓存的利用率的呢？这就涉及缓存行（Cache Line）的概念，Cache Line 是 CPU 缓存可操作的最小单位，CPU 缓存由若干个 Cache Line 组成。Cache Line 的大小与 CPU 架构有关，**在目前主流的 64 位架构下，Cache Line 的大小通常为 64 Byte**。CPU 在加载内存数据时，会将相邻的数据一同读取到 Cache Line 中，因为相邻的数据未来被访问的可能性最大，这样就可以避免 CPU 频繁与内存进行交互了。

伪共享问题是如何发生的呢？它又会造成什么影响呢？我们使用下面这幅图进行讲解。

![](https://static.cyub.vip/images/202206/false-sharing.png)

假设变量 A、B、C、D 被加载到同一个 Cache Line，它们会被高频地修改。当线程 1 在 CPU Core1 中中对变量 A 进行修改，修改完成后 CPU Core1 会通知其他 CPU Core 该缓存行已经失效。然后线程 2 在 CPU Core2 中对变量 C 进行修改时，发现 Cache line 已经失效，此时 CPU Core1 会将数据重新写回内存，CPU Core2 再从内存中读取数据加载到当前 Cache line 中。

由此可见，如果同一个 Cache line 被越多的线程修改，那么造成的写竞争就会越激烈，数据会频繁写入内存，导致性能浪费。

对于伪共享问题，我们应该如何解决呢？Disruptor 和 Mpsc Queue 都采取了空间换时间的策略，让不同线程共享的对象加载到不同的缓存行即可。下面我们通过一个简单的例子进行说明。

```java
public class FalseSharingPadding {
    protected long p1, p2, p3, p4, p5, p6, p7;
    protected volatile long value = 0;
    protected long p9, p10, p11, p12, p13, p14, p15;
}
```

![](https://static.cyub.vip/images/202206/cache-line-padding.png)

伪共享问题一般是非常隐蔽的，在实际开发的过程中，并不是项目中所有地方都需要花费大量的精力去优化伪共享问题。CPU Cache 的填充本身也是比较珍贵的，我们应该把精力聚焦在一些高性能的数据结构设计上，把资源用在刀刃上，使系统性能收益最大化。

## Lock Free vs Wait Free

## Reentrant Lock

Reentrant Lock(可重入锁)

## Thread-safe

Thread-safe(线程安全)

## 强一致性与弱一致性

一致性（Consistency）是指多副本（Replications）问题中的数据一致性。可以分为强一致性、弱一致性。

### 强一致性

强一致性也被可以被称做原子一致性（Atomic Consistency）或线性一致性（Linearizable Consistency），必须符合以下两个要求

    - 任何一次读都能立即读到某个数据的最近一次写的数据

    - 系统中的所有进程，看到的操作顺序，都和全局时钟下的顺序一致

简单地说就是假定对同一个数据集合，分别有两个线程 A、B 进行操作，假定 A 首先进行了修改操作，那么从时序上在 A 这个操作之后发生的所有 B 的操作都应该能立即（或者说实时）看到 A 修改操作的结果。

### 弱一致性

与强一致性相对的就是弱一致性，即数据更新之后，如果立即访问的话可能访问不到或者只能访问部分的数据。如果 A 线程更新数据后 B 线程经过一段时间后都能访问到此数据，则称这种情况为最终一致性，最终一致性也是弱一致性，只不过是弱一致性的一种特例而已

## 可见性与有序性

造成一致性的原因的根本原因是可见性与有序性。

## 参考资料

- [高性能无锁队列 Mpsc Queue](http://t.zoukankan.com/xiaojiesir-p-15562705.html)
- [Compare-and-swap](https://en.wikipedia.org/wiki/Compare-and-swap)
- [Java并发包提供了哪些并发工具类？](https://leeshengis.com/archives/9373)
- [一文澄清网上对 ConcurrentHashMap 的一个流传甚广的误解!](https://www.cnblogs.com/xiekun/p/16375705.html)
