# 概念

**CPU亲和性(affinity)描述了进程总是尽量在指定的CPU上运行，而不迁移到其他CPU的倾向性**。在多核CPU架构中，每个核心都有各自的L1和L2缓存，以及共用的L3缓存。如果进程频繁在多个CPU间迁移势必影响缓存命中率，并且还需要保存执行的上下文信息，进而导致调度效率低下。在Linux系统中，可以通过设置进程的CPU亲和性，将进程绑定到某CPU上面，保证进程只会在特定CPU上运行，避免迁移其他CPU过程中的低效性。

## 系统调用

Linux系统提供了两个系统调用`sched_setaffinity`和`sched_getaffinity`，可以分别用来设置和获取进程的CPU亲和性。

### 设置进程的CPU亲和性

`sched_setaffinity`系统调用函数原型是：

```c
int sched_setaffinity(pid_t pid, size_t cpusetsize, const cpu_set_t *mask);
```

各个参数说明：

- pid： 进程ID，即需要绑定特定CPU的进程ID
- cpusetsize: mask参数指向CPU位集合的大小
- mask: mask指向的是二进制位集合

mask参数是cpu_set_t类型参数，它是一个位图结构，每一个位代表一个CPU，当该位置为1则说明进程会绑定到该CPU上。下面示意图显示的是将进程绑定到CPU0，CPU1，CPU3，CPU4上：

![CPU亲和性](https://static.cyub.vip/images/202106/cpu_affinity.png)

### 获取进程的CPU亲和性

`sched_getaffinity`系统调用函数原型是：

```c
int sched_getaffinity(pid_t pid, size_t cpusetsize, cpu_set_t *mask);
```

参数就不在介绍了，跟上面设置亲和性函数一样。我们需要注意的是**进程默认情况下，可以在所有处理器间迁移调度运行，即默认情况下所有处理器对应的mask位都是1**，我们可以通过mask中位值为1的个数，计算出CPU总数。


### Go中使用CPU亲和性系统调用情况

Go中在runtime初始化过程中，会通过调用进程亲和性接口，获取CPU核数个数信息，从而创建相同数量的P。

```go
func osinit() {
	ncpu = getproccount() // 获取CPU个数
	physHugePageSize = getHugePageSize()
	osArchInit()
}

func getproccount() int32 {
	const maxCPUs = 64 * 1024
	var buf [maxCPUs / 8]byte
	r := sched_getaffinity(0, unsafe.Sizeof(buf), &buf[0]) // 获取当前进程亲和性
	if r < 0 {
		return 1
	}
	n := int32(0)
	for _, v := range buf[:r] {
		for v != 0 {
			n += int32(v & 1)
			v >>= 1
		}
	}
	if n == 0 {
		n = 1
	}
	return n
}

func schedinit() {
    ...
    procs := ncpu
	if n, ok := atoi32(gogetenv("GOMAXPROCS")); ok && n > 0 {
		procs = n
	}
	if procresize(procs) != nil {
		throw("unknown runnable goroutine during bootstrap")
	}
    ...
}
```

其中获取进程亲和性是通过汇编实现的：
```
TEXT runtime·sched_getaffinity(SB),NOSPLIT,$0
	MOVQ	pid+0(FP), DI
	MOVQ	len+8(FP), SI
	MOVQ	buf+16(FP), DX
	MOVL	$SYS_sched_getaffinity, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET
```


我们来测试一下：

```go
package main

import (
	"fmt"
	"unsafe"
)

func sched_getaffinity(int, uintptr, *byte) uint32 // 汇编函数声明

func main() {
	fmt.Println("proc count: ", getproccount())
}

func getproccount() int32 {
	const maxCPUs = 64 * 1024
	var buf [maxCPUs / 8]byte
	r := sched_getaffinity(0, unsafe.Sizeof(buf), &buf[0])
	if r < 0 {
		return 1
	}
	n := int32(0)
	for _, v := range buf[:r] {
		for v != 0 {
			n += int32(v & 1)
			v >>= 1
		}
	}
	if n == 0 {
		n = 1
	}
	return n
}
```

`sched_getaffinity`汇编实现： 
```
// sched_getaffinity.s

#include "textflag.h"
#define SYS_sched_getaffinity	204

TEXT ·sched_getaffinity(SB),NOSPLIT,$0
	MOVQ	pid+0(FP), DI
	MOVQ	len+8(FP), SI
	MOVQ	buf+16(FP), DX
	MOVL	$SYS_sched_getaffinity, AX
	SYSCALL
	MOVL	AX, ret+24(FP)
	RET
    
```
需要注意的是上面汇编代码最后一行需要是空行。


## 工作原理

Linux内核为每个CPU定义了一个类型为`struct rq`的可运行队列，正常情况下CPU只会从属于自己的可运行进程队列中选择一个进程来运行。所以设置进程CPU亲和性本质就是将进程放置到CPU的可运行进程队列即可，从而实现了进程绑定到某个CPU功能。具体详细内核代码，可以参考文末的参考链接。


## 参考

- [图解｜进程怎么绑定 CPU](https://mp.weixin.qq.com/s/mbyHJNVyN_0mV5efa2DAXg)