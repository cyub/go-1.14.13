## Cache Consistency

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

## Memory Barrier

为了解决MESI协议中引入`Store buffer`和`Invalidate queue`导致的并发问题，缓存系统采用内存屏障机制解决相关问题。

内存屏障分为读屏障(rmb)与写屏障(wmb)。**写屏障主要保证在写屏障之前的在Store buffer中的指令都真正的写入了缓存**。**读屏障主要保证了在读屏障之前所有Invalidate queue中所有的无效化指令都执行**。有了读写屏障的配合，那么在不同的核心上，缓存可以得到强同步。

在锁的实现上，一般lock都会加入读屏障，保证后续代码可以读到别的cpu核心上的未回写的缓存数据，而unlock都会加入写屏障，将所有的未回写的缓存进行回写。


## False Sharing

![](https://static.cyub.vip/images/202103/false_sharing.png)

当多线程需要修改互相独立的变量时，如果这些变量共享同一个缓存行，那么每个线程都要去竞争该缓存行的所有权来更新变量，并且获取所有权的线程更新完变量之后，未获取所有权线程就必须使自己对应的缓存行失效，然后从L3缓存中加载最新数据。

尽管看起来是不同变量在多个CPU内核之间共享了，但实际上却造成了性能降低，这就是为什么叫伪共享的原因。

## 资料

- [Why Memory Barriers？](http://www.wowotech.net/kernel_synchronization/Why-Memory-Barriers.html)
- [什么是内存屏障？ Why Memory Barriers ?](https://blog.csdn.net/s2603898260/article/details/109234770)
- [伪共享(False Sharing)](http://ifeve.com/falsesharing/)