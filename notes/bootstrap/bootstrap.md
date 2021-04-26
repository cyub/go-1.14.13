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

我们可以看到程序的入口函数是`_rt0_amd64`：
```
TEXT _rt0_amd64(SB),NOSPLIT,$-8
	MOVQ	0(SP), DI	// argc
	LEAQ	8(SP), SI	// argv
	JMP	runtime·rt0_go(SB)
```

`_rt0_amd64`函数做两件事，第一件是将程序启动命令参数信息argc，argv分别保存到DI和SI寄存器中，第二件事情是使用JMP指令跳到`runtime·rt0_go`函数。在Go汇编章节，我们介绍了JMP指令，它会隐式的更改IP寄存器值，使其指向JMP指令要跳转的函数入口地址，那么程序下次执行就会从该函数开始执行。

## g0初始化工作


Go运行时有两个全局变量m0和g0，分别代表着进程的主线程，以及主线程的系统调用栈。

```go
TEXT runtime·rt0_go(SB),NOSPLIT,$0
	// copy arguments forward on an even stack
	MOVQ	DI, AX		// argc
	MOVQ	SI, BX		// argv
	SUBQ	$(4*8+7), SP		// 2args 2auto
	ANDQ	$~15, SP
	MOVQ	AX, 16(SP)
	MOVQ	BX, 24(SP)

	// 初始化g0操作
	MOVQ	$runtime·g0(SB), DI // 将全局g0保存到DI寄存器中
	LEAQ	(-64*1024+104)(SP), BX
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