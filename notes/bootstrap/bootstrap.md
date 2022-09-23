# 调度器

我们将从下面代码入手分析，Go调度器初始化、主Goroutine初始化，调度等大致流程。

```go
package main

func main() {
	go func() {
		println("hello")
	}()

	println("world")
	for {
	}
}
```

## 程序入口

首先我们将上面代码构建成应用，然后使用GDB找到程序入口，开始调试分析之旅。

```bash
go build -gcflags="-N -l" -o hello hello.go
gdb ./hello #  使用gdb开始调试
```

进入gdb调试界面后，我们接着输入`info files`命令来获取程序相关信息，其中就包括程序入口地址，接下来打断点，然后运行起来应用。

```
(gdb) info files
Symbols from "/home/ubuntu/go-1.14.13/notes/bootstrap/hello".
Local exec file:
	`/home/ubuntu/go-1.14.13/notes/bootstrap/hello', file type elf64-x86-64.
	Entry point: 0x45cd80
	0x0000000000401000 - 0x000000000045ed36 is .text
	0x000000000045f000 - 0x000000000048bdb6 is .rodata
	0x000000000048bf40 - 0x000000000048c3e0 is .typelink
	0x000000000048c3e0 - 0x000000000048c3e8 is .itablink
	0x000000000048c3e8 - 0x000000000048c3e8 is .gosymtab
	0x000000000048c400 - 0x00000000004c7cd8 is .gopclntab
	0x00000000004c8000 - 0x00000000004c8020 is .go.buildinfo
	0x00000000004c8020 - 0x00000000004c9240 is .noptrdata
	0x00000000004c9240 - 0x00000000004cb3d0 is .data
	0x00000000004cb3e0 - 0x00000000004f86b0 is .bss
	0x00000000004f86c0 - 0x00000000004fd990 is .noptrbss
	0x0000000000400f9c - 0x0000000000401000 is .note.go.buildid
(gdb) b *0x45cd80
Breakpoint 1 at 0x45cd80: file /usr/local/go/src/runtime/rt0_linux_amd64.s, line 8.
(gdb) list /usr/local/go/src/runtime/rt0_linux_amd64.s:8
3	// license that can be found in the LICENSE file.
4
5	#include "textflag.h"
6
7	TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8
8		JMP	_rt0_amd64(SB)
```

程序的入口地址是**0x45cd80**，我们在入口地址处打上断点，需要注意的是给地址打断点需要在地址前面加上*号。查看入口地址处代码，我们可以看到程序的入口函数是`_rt0_amd64`：
```
TEXT _rt0_amd64(SB),NOSPLIT,$-8
	MOVQ	0(SP), DI	// argc
	LEAQ	8(SP), SI	// argv
	JMP	runtime·rt0_go(SB)
```

`_rt0_amd64`函数做两件事，第一件是将程序启动命令参数信息argc，argv分别保存到DI和SI寄存器中，第二件事情是使用JMP指令跳到`runtime·rt0_go`函数。在Go汇编章节，我们介绍了JMP指令，它会隐式的更改IP寄存器值，使其指向JMP指令要跳转的函数入口地址，那么程序下次执行就会从该函数开始执行。

argc 和 argv分别是程序启动的参数个数和参数数组。这个和c程序中main函数启动时候的argc和argv是一致的。

```c
include <stdio.h>

int main(int argc, char *argv[])
{
    int i;
    for (i=0; i < argc; i++)
        printf("Argument %d is %s.\n", i, argv[i]);

    return 0;
}
```

借助GDB验证下，我们验证下这两个参数的值：
```
(gdb) list runtime/asm_amd64.s:17
12	// kernel for an ordinary -buildmode=exe program. The stack holds the
13	// number of arguments and the C-style argv.
14	TEXT _rt0_amd64(SB),NOSPLIT,$-8
15		MOVQ	0(SP), DI	// argc
16		LEAQ	8(SP), SI	// argv
17		JMP	runtime·rt0_go(SB)
18
19	// main is common startup code for most amd64 systems when using
20	// external linking. The C startup code will call the symbol "main"
21	// passing argc and argv in the usual C ABI registers DI and SI.
(gdb) b runtime/asm_amd64.s:17
Breakpoint 2 at 0x459869: file /usr/local/go/src/runtime/asm_amd64.s, line 17.
(gdb) c
Continuing.

Breakpoint 2, _rt0_amd64 () at /usr/local/go/src/runtime/asm_amd64.s:17
17		JMP	runtime·rt0_go(SB)
(gdb) p $rdi
$1 = 1
(gdb) x /xg $rsi
0x7fffffffe418:	0x00007fffffffe672
(gdb) x /s 0x00007fffffffe672
0x7fffffffe672:	"/home/ubuntu/go-1.14.13/notes/bootstrap/hello"
```

我们可以看到rdi寄存器存储的参数个数确实是1，rsi指向的参数数组，其第一个参数确实是启动命令。

## g0初始化工作

Go运行时有两个全局变量m0和g0，分别代表着进程的主线程，以及主线程的系统调用栈。

```go
TEXT runtime·rt0_go(SB),NOSPLIT,$0
	// copy arguments forward on an even stack
	MOVQ	DI, AX		// argc
	MOVQ	SI, BX		// argv
	SUBQ	$(4*8+7), SP // 2args 2auto; 其中2*8栈空间用来临时保存argc和argv参数的，另外2*8空间用来存放后面runtime·args函数的参数(callee的参数栈空间是由caller提供的)
	ANDQ	$~15, SP // 等效于 and rsp,0xfffffffffffffff0; $~15表示的是将15按位取反后的立即数，然后和SP寄存器值进行与运算，最后将寄存器值保存到SP寄存器中。
	// 这么操作目的保证SP地址是16位对齐的，也可以说成保证SP地址是16的倍数。
	MOVQ	AX, 16(SP)
	MOVQ	BX, 24(SP)

	// 初始化g0操作
	MOVQ	$runtime·g0(SB), DI // 将全局g0保存到DI寄存器中
	LEAQ	(-64*1024+104)(SP), BX // g0栈空间大小为64k-104
	MOVQ	BX, g_stackguard0(DI) // 设置全局g0的stackguard0和stackguard1
	MOVQ	BX, g_stackguard1(DI)
	MOVQ	BX, (g_stack+stack_lo)(DI) // 设置全局g0的lo和hi
	MOVQ	SP, (g_stack+stack_hi)(DI)
```

上面代码中完成g0的初始化，我们继续往下看调度器初始化工作代码。

## 调度器初始化、main groutine创建以及调度运行

```go
    LEAQ	runtime·m0+m_tls(SB), DI // 将m0.tls保存到DI寄存器中，即DI = &m0.tls[0]
	CALL	runtime·settls(SB) // 设置本地线程存储(thread local store)，最终结果是使段寄存器fs存储的是m0.tls[1]的地址

    // 存储信息到本地线程存储中，然后从其中取出来，检验是否一致，若不一致则调用runtime·abort终止程序执行
	get_tls(BX)
	MOVQ	$0x123, g(BX)
	MOVQ	runtime·m0+m_tls(SB), AX
	CMPQ	AX, $0x123
	JEQ 2(PC)
	CALL	runtime·abort(SB)

	get_tls(BX) // 将段寄存器存储的地址，即m0.tls[1]的地址取出来，保存到BX寄存器中
	LEAQ	runtime·g0(SB), CX // runtime.g0地址保存到CX寄存器中，即CX = &runtime.g0
	MOVQ	CX, g(BX) // g(BX) 是m0.tls[0]，此处指令完成工作等效于 m0.tls[0] = &runtime.g0
	LEAQ	runtime·m0(SB), AX // AX = &runtime.m0

	// save m->g0 = g0
	MOVQ	CX, m_g0(AX) // m.g0 = &runtime.g0
	// save m0 to g0->m
	MOVQ	AX, g_m(CX) // g0.m = &runtime.m0

	CLD				// convention is D is always left cleared
	CALL	runtime·check(SB) // 执行类型大小检查，atomic.Cas检查等工作

	MOVL	16(SP), AX		// copy argc
	MOVL	AX, 0(SP)
	MOVQ	24(SP), AX		// copy argv
	MOVQ	AX, 8(SP)
	CALL	runtime·args(SB) // 命令行参数与环境变量处理工作
	CALL	runtime·osinit(SB) // 主要是利用sched_getaffinity系统调用获取CPU核数来初始化全局变量ncpu
	CALL	runtime·schedinit(SB) // 调度器初始化工作

	// create a new goroutine to start program
	MOVQ	$runtime·mainPC(SB), AX		// $runtime·mainPC是一个二级地址，指向runtime.main
	PUSHQ	AX // 将AX入栈，即&runtime.main入栈。此时&runtime.main作为newproc第二个参数
	PUSHQ	$0			// 0 作为newproc第二个参数
	CALL	runtime·newproc(SB) // 调用newproc 用来创建goroutine
	POPQ	AX
	POPQ	AX

	// start this M
	CALL	runtime·mstart(SB) // 执行调度逻辑

	CALL	runtime·abort(SB)	// 正常情况下，mstart相当于死循环调度，永远不会执行到此处。
```

上面代码包含调度器大致流程，流程有点多，我们来梳理一下：

1. 完成线程本地化存储

    - fs段寄存器存储的是m0.tls[1]地址， m0.tls[1]存储的g0的地址。通过fs段寄存器可以“全局”地访问到m

2. 完成m0和g0的关联

    - m0.g0 = g0
    - g0.m = m0

3. 执行类型检查，命令行参数初始化，cpu个数初始化，以及调度器初始化工作

4. 调用runtime.newproc，根据runtime.main函数生成第一个goroutine，即main goroutine。

5. 调用runtime.mstart循环执行调度流程


### 本地化存储

本地化存储是由汇编实现的：

```go
// runtime/sys_linux_amd64.s
#define SYS_arch_prctl		158

TEXT runtime·settls(SB),NOSPLIT,$32
#ifdef GOOS_android
	// Android stores the TLS offset in runtime·tls_g.
	SUBQ	runtime·tls_g(SB), DI
#else
	ADDQ	$8, DI	// DI保存是m0.tls[0]地址, 现在将该地址加8，此后DI保存的是m0.tls[1]的地址。
#endif
	MOVQ	DI, SI // arch_prctl系统调用的第二个参数是m0.tls[1]的地址
	MOVQ	$0x1002, DI	// arch_prctl系统调用的第一个参数是：ARCH_SET_FS
	MOVQ	$SYS_arch_prctl, AX
	SYSCALL // 调用系统调用arch_prctl
	CMPQ	AX, $0xfffffffffffff001
	JLS	2(PC)
	MOVL	$0xf1, 0xf1  // crash
	RET
```

上面汇编中通过arch_prctl系统调用设置段寄存器fs指向了m0.tls[1]，本地变量的读取是通过get_tls(BX)和g(BX)来读取。

```
get_tls(BX)
MOVQ $0x123, g(BX)
```

注意是get_tls是伪汇编代码，只是提示编译器开始使用本地存储。上面go汇编代码最终的实际汇编如下：

```
movq   $0x123, %fs:0xfffffffffffffff8
```

`%fs:0xfffffffffffffff8` 等价于%fs - 8（-8的补码是0xfffffffffffffff8），由于fs存储的是m0.tls[1], 那么`%fs:0xfffffffffffffff8`指向的就是m0.tls[0]。



### 调度器初始化

接下来我们来探究下调度器初始化代码，这部分代码由Go 语言实现：

```go
func schedinit() {
	_g_ := getg() // 将当前本地线程存储的g保存到临时变量_g_中

	sched.maxmcount = 10000 // maxmcount记录调度器运行创建的M个数，默认是10000

	tracebackinit()
	moduledataverify()
	stackinit() // 栈缓存池初始化
	mallocinit() // 内存分配器初始化工作
	fastrandinit() // 随机数种子初始化工作，主要是读取/dev/urandom文件完成
	mcommoninit(_g_.m) // m初始化工作
	cpuinit()       // cpu初始化工作，主要完成GODEBUG环境变量的解析处理和完成cpu特性开启关闭
	alginit()       // maps must not be used before this call
    // 符号表相关的初始化
	modulesinit()   // provides activeModules
	typelinksinit() // uses maps, activeModules
	itabsinit()     // uses activeModules

	msigsave(_g_.m)
	initSigmask = _g_.m.sigmask

	goargs() // 将命令行参数保存到全局变量argslice中
	goenvs() // 将环境变量保存到全局变量envs中
	parsedebugvars() // 解析GODEBUG参数到全局debug结构体中
	gcinit() // gc初始化

	sched.lastpoll = uint64(nanotime())
	procs := ncpu
	if n, ok := atoi32(gogetenv("GOMAXPROCS")); ok && n > 0 {
		procs = n
	}
	if procresize(procs) != nil { // 初始化全局变量allp。所有的P都存放在allp这个全局变量中
		throw("unknown runnable goroutine during bootstrap")
	}

	...
}
```

#### m初始化工作

在调度器初始化工作时顺便也完成m的通用初始化工作。

```go

func mcommoninit(mp *m) {
	_g_ := getg()

	if _g_ != _g_.m.g0 { // 系统调用栈帧信息，用户态不需关心，故不记录
		callers(1, mp.createstack[:]) // 记录用户态g的栈帧信息
	}

	lock(&sched.lock) // m初始化时候加锁
	if sched.mnext+1 < sched.mnext {
		throw("runtime: thread ID overflow")
	}
	mp.id = sched.mnext // shed.mnext作为m的id
	sched.mnext++ // sched.mnext记录着创建m的数量（每个m创建之后都会调用mcommoninit完成初始化工作）
	checkmcount() // 检查当前m数量，当前m数量不能超过sched.maxmcount

    // rand相关的初始化操作
	mp.fastrand[0] = uint32(int64Hash(uint64(mp.id), fastrandseed))
	mp.fastrand[1] = uint32(int64Hash(uint64(cputicks()), ^fastrandseed))
	if mp.fastrand[0]|mp.fastrand[1] == 0 {
		mp.fastrand[1] = 1
	}

	mpreinit(mp) // 创建32k大小g，并赋值给m.gsignal，用来处理信号。并且设置m.gsignal.m = m 将m和g关联起来
	if mp.gsignal != nil {
		mp.gsignal.stackguard1 = mp.gsignal.stack.lo + _StackGuard
	}

    // mp.alllink指向全局变量allm，allm是链表，存放所有的m
	mp.alllink = allm
    // allm = mp，这一步和上面一步操作相当于在allm链表头部插入mp
	atomicstorep(unsafe.Pointer(&allm), unsafe.Pointer(mp))
	unlock(&sched.lock)

	// Allocate memory to hold a cgo traceback if the cgo call crashes.
	if iscgo || GOOS == "solaris" || GOOS == "illumos" || GOOS == "windows" {
		mp.cgoCallers = new(cgoCallers)
	}
}
```

### main goroutine的创建

```
// create a new goroutine to start program
MOVQ	$runtime·mainPC(SB), AX		// $runtime·mainPC是一个二级地址，指向runtime.main
PUSHQ	AX // 将AX入栈，即&runtime.main入栈。此时&runtime.main作为newproc第二个参数
PUSHQ	$0			// 0 作为newproc第二个参数
CALL	runtime·newproc(SB) // 调用newproc 用来创建goroutine
```