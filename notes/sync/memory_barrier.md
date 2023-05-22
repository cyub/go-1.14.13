## Cache Consistency

![](https://static.cyub.vip/images/202107/cpu_cache.png)

为了解决CPU与内存访问之间的巨大速度差异，在CPU中引入Cache系统，每个CPU核心都有自己独有的Cache，但这会造成同一个内存块，在多个CPU之间有多个备份，这就会造成访问数据的不一致。为了解决缓存不一致的情况，需要采用一致性协议`MESI`进行同步。

为了解决`MESI`协议中，Shared或Invalidate状态下的`Cache line`需要等待其他CPU的invalidate acknowledge导致的不必要的等待，引入了`Store buffer`；为了解决CPU响应invalidate message不及时问题，引入了`Invalidate queue`。由于`Store buffer`和`Invalidate queue`的引入，所以最终的缓存架构如下：

![](https://static.cyub.vip/images/202103/memory_arch.gif)

### MESI缓存一致性协议

**缓存系统中是以缓存行（cache line）为单位存储的**。缓存行是2的整数幂个连续字节，一般为32-256个字节。最常见的缓存行大小是64个字节。

`MESI`协议是`Cache line`四种状态的首字母的缩写，分别是**修改（Modified）态、独占（Exclusive）态、共享（Shared）态和失效（Invalid）态**。Cache 中缓存的每个`Cache Line`都必须是这四种状态中的一种。

- Modified

    表示当前cache line的数据项被修改过（未保存至内存），这种状态下该数据是被该CPU占有的，其他任何CPU中都不会存在有效的相同数据项

- Exclusive

    表示该数据被本CPU拥有，在其他CPU的缓存中都不存在对应的数据，这种情况下该CPU可以直接修改该数据项，而不需要发送消息

- Shared

    表示该数据被多个CPU所共享（存在于多个CPU缓存中），这种情况下CPU不能直接修改该数据项

- Invalid

    表示该数据无效，即删除，CPU加载该数据时不需要重新加载，不能直接使用


#### MESI协议状态转移

根据protocol message的发送和接收情况，`cache line`会在`modified`, `exclusive`, `shared`, 和`invalid`这四个状态之间迁移，具体如下图所示:

![MESI协议状态转移](https://static.cyub.vip/images/202103/mesi_state_transfer.gif)

- M->E

    cache可以通过write back transaction将一个cache line的数据写回到memory中（或者下一级cache中），这时候该cache line的状态从Modified迁移到Exclusive状态。对于cpu而言，cache line中的数据仍然是最新的，而且是该cpu独占的，因此可以不通知其他cpu cache而直接修改之

- E->M

    在Exclusive状态下，cpu可以直接将数据写入cache line，不需要其他操作。相应的，该cache line状态从Exclusive状态迁移到Modified状态。这个状态迁移过程不涉及bus上的Transaction（即无需MESI Protocol Messages的交互）。

- M->I

    CPU 在总线上收到一个read invalidate的请求，同时，该请求是针对一个处于modified状态的cache line，在这种情况下，CPU必须该cache line状态设置为无效，并且用read response和invalidate acknowledge来回应收到的read invalidate的请求，完成整个bus transaction。一旦完成这个transaction，数据被送往其他cpu cache中，本地的copy已经不存在了

- I->M

    CPU需要执行一个原子的readmodify-write操作，并且其cache中没有缓存数据，这时候，CPU就会在总线上发送一个read invalidate用来请求数据，同时想独自霸占对该数据的所有权。该CPU的cache可以通过read response获取数据并加载cache line，同时，为了确保其独占的权利，必须收集所有其他cpu发来的invalidate acknowledge之后（其他cpu没有local copy），完成整个bus transaction

- S->M

    CPU需要执行一个原子的readmodify-write操作，并且其local cache中有read only的缓存数据（cache line处于shared状态），这时候，CPU就会在总线上发送一个invalidate请求其他cpu清空自己的local copy，以便完成其独自霸占对该数据的所有权的梦想。同样的，该cpu必须收集所有其他cpu发来的invalidate acknowledge之后，才算完成整个bus transaction

- M->S

    在本cpu独自享受独占数据的时候，其他的cpu发起read请求，希望获取数据，这时候，本cpu必须以其local cache line的数据回应，并以read response回应之前总线上的read请求。这时候，本cpu失去了独占权，该cache line状态从Modified状态变成shared状态（有可能也会进行写回的动作）

- E->S

    这个迁移和f类似，只不过开始cache line的状态是exclusive，cache line和memory的数据都是最新的，不存在写回的问题。总线上的操作也是在收到read请求之后，以read response回应。

- S->E

    如果cpu认为自己很快就会启动对处于shared状态的cache line进行write操作，因此想提前先霸占上该数据。因此该cpu会发送invalidate敦促其他cpu清空自己的local copy，当收到全部其他cpu的invalidate acknowledge之后，transaction完成，本cpu上对应的cache line从shared状态切换exclusive状态。还有另外一种方法也可以完成这个状态切换：当所有其他的cpu对其local copy的cache line进行写回操作，同时将cache line中的数据设为无效（主要是为了为新的数据腾些地方），这时候，本cpu坐享其成，直接获得了对该数据的独占权。

- E->I

    其他的CPU进行一个原子的read-modify-write操作，但是数据在本cpu的cache line中，因此其他的那个CPU会发送read invalidate，请求对该数据以及独占权。本cpu回送read response”和“invalidate acknowledge”，一方面把数据转移到其他cpu的cache中，另外一方面，清空自己的cache line。

- I->E

    cpu想要进行write的操作但是数据不在local cache中，因此该cpu首先发送了read invalidate启动了一次总线transaction。在收到read response回应拿到数据，并且收集所有其他cpu发来的invalidate acknowledge之后（确保其他cpu没有local copy），完成整个bus transaction。当write操作完成之后，该cache line的状态会从Exclusive状态迁移到Modified状态。

- I->S

    本CPU执行读操作，发现local cache没有数据，因此通过read发起一次bus transaction，来自其他的cpu local cache或者memory会通过read response回应，从而将该cache line从Invalid状态迁移到shared状态。

- S->I

    当cache line处于shared状态的时候，说明在多个cpu的local cache中存在副本，因此这些cache line中的数据都是read only的，一旦其中一个cpu想要执行数据写入的动作，必须先通过invalidate获取该数据的独占权，而其他的CPU会以invalidate acknowledge回应，清空数据并将其cache line从shared状态修改成invalid状态。

#### Store buffer的引入

从上面状态转换中可以看到当CPU的某一`cache line`处于Modify或者exclusive状态时，store指令会将数据很快写入`cache line`内，但是当CPU的`cache line`处于Shared或者Invalidate状态时，srote指令写入数据之前必须等待其他CPU的响应。

为了避免这种不必要的等待，CPU内引入了`store buffer`即当CPU需要等待响应时，现在可以直接将数据写入`store buffer`，然后继续执行下一条指令,等到收到响应后，再将`store buffer`中的数据写入`cache line`。

引入Store buffer会导致一系列并发问题。针对这类问题解决办法是`Store Forwarding`和`Memory Barrier`。

##### Store Forwarding

```c
a=1
b=a+1
assert(b == 2)
```

假设a， b被初始化为0，a在CPU1的某一cache line上，b在CPU0的另一cache line上，执行顺序如下：

1. CPU0开始执行a=1,
2. CPU0 cache missing，CPU0发送read invalidate，为了获得cache line（包含a）的独有权限
3. CPU0将a=1存储到store buffer
4. CPU1收到read invalidate, 发送a=1 read respond，无效cache line，发送invalidate ack
5. CPU0执行b=a+1=1
6. CPU0收到响应，load a=0 到cache line
7. CPU0 将store buffer内数据存储到 cache line (a=1)
8. CPU0 执行assert失败

出现上面问题主要是数据a信息，一份在cache line中，一份在store buffer中。为了解决这个问题会在硬件层面采用`store forwarding`策略。该策略会在执行load操作时，优先从store buffer中读取，如果能够读取到数据，则直接返回，如果没有才会在cache line中读取。

##### Memory Barrier

```c
void foo(void)
{
    a = 1;
    b = 1;
}
 
void bar(void)
{
    while (b == 0) continue;
    assert(a == 1);
}
```

假设a和b初始化值都是0，CPU0执行foo函数，CPU1执行bar函数。a变量在CPU1的cache中，b在CPU0 cache中，执行的操作顺序如下：

1.  CPU0执行a=1的赋值操作，由于a不在local cache中，因此，CPU0将a值放到store buffer中之后，发送了read invalidate命令到总线上去

2. CPU1执行 while (b == 0) 循环，由于b不在CPU1的cache中，因此，CPU1发送一个read message到总线上，看看是否可以从其他cpu的local cache中或者memory中获取数据

3. CPU0继续执行b=1的赋值语句，由于b就在自己的local cache中（cache line处于modified状态或者exclusive状态），因此CPU0可以直接操作将新的值1写入cache line。

4.  CPU0收到了read message，将最新的b值”1“回送给CPU1，同时将b cache line的状态设定为shared

5. CPU1收到了来自CPU0的read response消息，将b变量的最新值”1“值写入自己的cache line，状态修改为shared

6. 由于b值等于1了，因此CPU1跳出while (b == 0)的循环，继续前行。

7. CPU1执行assert(a == 1)，这时候CPU1的local cache中还是旧的a值，因此assert(a == 1)失败

8. CPU1收到了来自CPU0的read invalidate消息，以a变量的值进行回应，同时清空自己的cache line，但是这已经太晚了

9. CPU0收到了read response和invalidate ack的消息之后，将store buffer中的a的最新值”1“数据写入cache line，然而CPU1已经assertion fail了

上面问题原因是foo中b已经赋值了，但是a还没有，最终导致bar断言失败。对于这种问题CPU无法解决，因为其无法知道变量之间的关系。对于这种问题，CPU提供了memory barrier指令，让软件告诉CPU这类的关系。

```c
void foo(void)
{
    a = 1;
    smp_mb();
    b = 1;
}
```

smp_mb()指令可以迫使CPU在进行后续store操作前刷新store-buffer。这样就可以保证在执行b=1的时候CPU0-store-buffer中的a已经刷新到cache中了，此时CPU1-cache中的a 必然已经标记为invalid。对于CPU1中执行的代码，则可以保证当b==0为假时，a已经不在CPU1-cache中，从而必须从CPU0- cache传递，得到新值“1”。


#### Invalidate queue的引入

Invalidate queue用于缓存Shared->Invalid状态的指令，当cpu收到其它核心的RFO指令后，会将自身对应的cache line无效化，但是当核心比较忙的时候，无法立刻处理，所以引入Invalidate queue，当收到RFO指令后，立刻回应，将无效化的指令投入Invalidate queue。

上面`Store buffer`引入的问题2的示例中，并没有解决由于引入`Invalidate queue`导致的并发问题。我们需要在bar函数引入内存屏障，让所有无效指令都必须先执行（即数据a的无效指令一定要在读取a之前先执行)。

```c
void bar(void)
{
    while (b == 0) continue;
    smp_mb();
    assert(a == 1);
}
```

smp_mb();既可以用来处理storebuffer中的数据，也可以用来处理Invalidation Queue中的Invalid消息。实际上，memory barrier确实可以细分为“write memory barrier(wmb)”和“read memory barrier(rmb)”。rmb只处理Invalidate Queues，wmb只处理store buffer。

所以最佳解决方案是：

```c
void foo(void)
{
    a = 1;
    smp_wmb();/*CPU1要使用该值，因此需要及时更新处理store buffer中的数据*/
    b = 1;
}
 
void bar(void)
{
    while (b == 0) continue;
    smp_rmb();/*由于CPU0修改了a值，使用此值时及时处理Invalidation Queue中的消息*/
    assert(a == 1);
}
```

## Memory Barrier（内存屏障)

程序在运行时内存实际的访问顺序和程序代码编写的访问顺序不一定一致，这就是内存乱序访问。内存乱序访问行为出现的理由是为了提升程序运行时的性能。内存乱序访问主要发生在两个阶段：

- 编译时，编译器优化导致内存乱序访问，也就是**指令重排**
- 运行时，多 CPU 间交互引起**内存乱序**访问

Memory Barrier 能够让 CPU 或编译器在内存访问上有序。一个 Memory Barrier 之前的内存访问操作必定先于其之后的完成。Memory Barrier 包括两类：

- CPU Memory Barrier
- 编译器Memory Barrier

内存屏障的维基百科定义：

> A memory barrier, also known as a membar, memory fence or fence instruction, is a type of barrier instruction that causes a central processing unit (CPU) or compiler to enforce an ordering constraint on memory operations issued before and after the barrier instruction.

翻译过来就是：内存屏障是用来强制约束屏障指令前后内存操作顺序。现代 CPU 会采用乱序执行、流水线、分支预测、多级缓存等方法提高性能，但这些方法同时也影响了指令执行和生效的次序。这种乱序对单线程执行是无影响的，执行的结果与原指令顺序执行保持一致，但在多线程环境下则会造成非预期的行为

### CPU 内存屏障

在CPU乱序执行时，一个处理器真正执行指令的顺序由可用的输入数据决定，而非程序员编写的顺序。早期的处理器为有序处理器（In-order processors），有序处理器处理指令通常有以下几步：

1. 指令获取
2. 如果指令的输入操作对象（input operands）可用（例如已经在寄存器中了），则将此指令分发到适当的功能单元中。如果一个或者多个操作对象不可用（通常是由于需要从内存中获取），则处理器会等待直到它们可用
3. 指令被适当的功能单元执行
4. 功能单元将结果写回寄存器堆（Register file，一个 CPU 中的一组寄存器）

相比之下，乱序处理器（Out-of-order processors）处理指令通常有以下几步：

1. 指令获取
2. 指令被分发到指令队列
3. 指令在指令队列中等待，直到输入操作对象可用（一旦输入操作对象可用，指令就可以离开队列，即便更早的指令未被执行）
4. 指令被分配到适当的功能单元并执行
5. 执行结果被放入队列（而不立即写入寄存器堆）
6. 只有所有更早请求执行的指令的执行结果被写入寄存器堆后，指令执行的结果才被写入寄存器堆（执行结果重排序，让执行看起来是有序的）

从上面的执行过程可以看出，乱序执行相比有序执行能够避免等待不可用的操作对象（有序执行的第二步）从而提高了效率。现代的机器上，处理器运行的速度比内存快很多，有序处理器花在等待可用数据的时间里已经可以处理大量指令了。

从上面的乱序处理器处理指令的过程，我们能得到几个结论：

1. 对于单个 CPU 指令获取是有序的（通过队列实现）
2. 对于单个 CPU 指令执行结果也是有序返回寄存器堆的（通过队列实现）

也就是说UP(uniprocessor)系统下不存在CPU交互导致的内存乱序访问了，此时只需要考虑编译器导致的内存乱序访问。

为了解决MESI协议中引入`Store buffer`和`Invalidate queue`导致的并发问题，缓存系统采用内存屏障机制解决相关问题。

内存屏障分为读屏障(rmb)与写屏障(wmb)。**写屏障主要保证在写屏障之前的在Store buffer中的指令都真正的写入了缓存**。**读屏障主要保证了在读屏障之前所有Invalidate queue中所有的无效化指令都执行**。有了读写屏障的配合，那么在不同的核心上，缓存可以得到强同步。

在锁的实现上，一般lock都会加入读屏障，保证后续代码可以读到别的cpu核心上的未回写的缓存数据，而unlock都会加入写屏障，将所有的未回写的缓存进行回写。

内存屏障(memory barrier)，又叫做内存栅栏(memory fence)，对于X86平台来说，内存屏障分为三种：

- Load Barrier，读屏障

    对应是lfence指令，在读指令前插入读屏障，可以让高速缓存中的数据失效，重新从主内存加载数据。对应的代码为：

    __asm__ __volatile__("lfence" : : : "memory");

- Store Barrier，写屏障

    对应是sfence指令，在写指令之后插入写屏障，能让写入缓存的最新数据写回到主内存。对应的代码为：

    __asm__ __volatile__("sfence" : : : "memory");

- Full Barrier，读写屏障

    对应是mfence指令，具备lfence和sfence的能力。

    __asm__ __volatile__("mfence" : : : "memory");

除了内存屏障外，Lock前缀指令能够完成类似内存屏障的功能。Lock会对CPU总线和高速缓存加锁，可以理解为CPU指令级的一种锁。它后面可以跟ADD, ADC, AND, BTC, BTR, BTS, CMPXCHG, CMPXCH8B, DEC, INC, NEG, NOT, OR, SBB, SUB, XOR, XADD, and XCHG等指令。

Lock前缀实现了类似的能力：

- 它先对总线/缓存加锁，然后执行后面的指令，最后释放锁后会把高速缓存中的脏数据全部刷新回主内存

- 在Lock锁住总线的时候，其他CPU的读写请求都会被阻塞，直到锁释放。Lock后的写操作会让其他CPU相关的cache line失效，从而从新从内存加载最新的数据。这个是通过缓存一致性协议做的

### 编译器内存屏障

编译器出于对代码做出优化的考虑，可能改变实际执行指令的顺序(这叫**指令重排**)，如果程序的逻辑依赖内存变量的访问顺序，那这时候就有可能会造成意向不到的后果。上面例子稍微改一下：

```c
// instruction_reorder.c
#include <assert.h>

int a = 0, b = 0, c = 1;

void foo(void)
{
    a = c;
    b = 1;
}

void bar(void)
{
    while (b == 0) continue;
    assert(a == 1);
}
```

首先我们不进行任何优化进行编译得到foo函数核心汇编代码：

```assembly
// gcc -S instruction_reorder.c -o instruction_reorder.s

movl	c(%rip), %eax
movl	%eax, a(%rip) // a = 1
movl	$1, b(%rip) // b = 1
```

开启O2级别优化：

```assembly
// gcc -O2 -S instruction_reorder.c -o instruction_reorder_o2.s
movl	c(%rip), %eax
movl	$1, b(%rip) // b = 1
movl	%eax, a(%rip) // a = 1
```

从上面可以看到当开启优化之后，发生了指令重排，导致b先被复制为1, a再接着赋值。如果此时另一个线程在a赋值之前执行bar时候，就会导致断言失败。


### UP(单核)和SMP(多核)下的内存屏障

Linux内核源码中支持的屏障如下：

```c
#define barrier() __asm__ __volatile__("" : : : "memory");

#ifdef CONFIG_SMP // SMP系统
#define smp_mb()    mb()
#define smp_rmb()   rmb()
#define smp_wmb()   wmb()

#else // UP系统

#define smp_mb()    barrier() // 只需要编译器屏障既可以。
#define smp_rmb()   barrier()
#define smp_wmb()   barrier()

#endif

#ifdef CONFIG_X86_32

#define mb() alternative("lock; addl $0,0(%%esp)", "mfence", X86_FEATURE_XMM2) // 32位操作系统，不一定有xfence指令(Pentium4引入)，那就使用lock前缀指令
#define rmb() alternative("lock; addl $0,0(%%esp)", "lfence", X86_FEATURE_XMM2)
#define wmb() alternative("lock; addl $0,0(%%esp)", "sfence", X86_FEATURE_XMM)

#else

#define mb()    asm volatile("mfence":::"memory")
#define rmb()   asm volatile("lfence":::"memory")
#define wmb()   asm volatile("sfence" ::: "memory")

#endif
```

从上面可以看到UP系统只有编译器屏障。32位SMP系统使用lock前缀指令做CPU内存屏障，64位SMP系统使用xfench做CPU内存屏障。


## Cache Line

对于CPU来说，它并不关心程序中操作的对象，它只关心对某个内存块的读和写，为了让读写速度更快，CPU会首先把数据从主存中的数据以cache line的粒度读到CPU cache中，一个cache line一般是64 bytes。假设程序中读取某一个int变量，CPU并不是只从主存中读取4个字节，而是会一次性读取64个字节，然后放到cpu cache中。因为往往紧挨着的数据，更有可能在接下来会被使用到。比如遍历一个数组，因为数组空间是连续的，所以并不是每次取数组中的元素都要从主存中去拿，第一次从主存把数据放到cache line中，后续访问的数据很有可能已经在cache中了。

### cache hit

CPU获取的内存地址在cache中存在，叫做cache hit。

### cache miss

如果CPU的访问的内存地址不在L1 cache中，就叫做L1 cache miss，由于访问主存的速度远远慢于指令的执行速度，一旦发生cache miss，CPU就会在上一级cache中获取，最差的情况需要从主存中获取。一旦要从主存中获取数据，当前指令的执行相对来说就会显得非常慢。

## False Sharing

![](https://static.cyub.vip/images/202103/false_sharing.png)

当多线程需要修改互相独立的变量时，如果这些变量共享同一个缓存行，那么每个线程都要去竞争该缓存行的所有权来更新变量，并且获取所有权的线程更新完变量之后，未获取所有权线程就必须使自己对应的缓存行失效，然后从L3缓存中加载最新数据。

尽管看起来是不同变量在多个CPU内核之间共享了，但实际上却造成了性能降低，这就是为什么叫伪共享的原因。

## 资料

- [Why Memory Barriers？](http://www.wowotech.net/kernel_synchronization/Why-Memory-Barriers.html)
- [什么是内存屏障？ Why Memory Barriers ?](https://blog.csdn.net/s2603898260/article/details/109234770)
- [伪共享(False Sharing)](http://ifeve.com/falsesharing/)
- [Data alignment: Straighten up and fly right](https://developer.ibm.com/technologies/systems/articles/pa-dalign/)
- [Memory Barriers:  a Hardware View for Software Hackers](http://www.puppetmastertrading.com/images/hwViewForSwHackers.pdf)
- [一文讲解，Linux内核——Memory Barrier（内存屏障）](https://zhuanlan.zhihu.com/p/498449295)