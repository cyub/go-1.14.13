# Golang 源码分析

Golang版本是go1.14.13

## 目录

- 运行流程
	- [ ] [启动流程](./notes/bootstrap/bootstrap.md)
- 堆栈
	- [ ] [Goroutine栈设计](./notes/go-stack.md)
- 错误处理
	- [ ] [panic-defer-recover](./notes/error/panic.md)
- sync
	- [x] [Map](./notes/sync/map.md)
	- [x] [Waitgroup](./notes/sync/waitgroup.md)
	- [x] [atomic](./notes/sync/atomic.md)
	- [x] [Once](./notes/sync/once.md)
	- [ ] [Pool](./notes/sync/pool.md)
	- [ ] [Mutex](./notes/sync/mutex.md)
	- [ ] [Cond](./notes/sync/cond.md)
	- [x] [同步机制](./notes/sync/method.md)
- 内存管理
	- [ ] [内存分配管理](./notes/memory/memory_allocator.md)
	- [ ] [逃逸分析](./notes/misc/escape-analysis.md)
	- [ ] [GC](./notes/memory/gc.md)
	- [x] [缓存一致性、内存屏障、伪共享](./notes/sync/memory_barrier.md)
- slice
	- [ ] [slice](./notes/slice/slice.md)
- channel
	- [x] [channel](./notes/channel/channel.md)
	- [ ] [select选择器](./notes/channel/select.md)
- map
	- [ ] [map](./notes/map/map.md)
- type system
	- [ ] [type system](./notes/type/type.md)
	- [ ] [interface](./notes/type/interface.md)
	- [ ] [reflect](./notes/type/reflect.md)
	- [x] [Function Value、Closure、Method](./notes/misc/function_closure_method.md)
	- [x] [值传递、引用传递、值类型变量、引用类型变量](./notes/misc/pass_by_value.md)
	- [x] [unsafe](./notes/type/unsafe.md)
	- [ ] [函数是一等公民](./note/type/first_class.md)
	- [ ] [字符串](./note/type/string.md)
	- [ ] [数组](./note/type/array.md)
	- [ ] [切片](./note/type/slice.md)
- [x] [context](./notes/context/context.md)
- [ ] [syscall](./notes/syscall/syscall.md)
- misc
	- [x] [no copy机制](./notes/misc/nocopy.md)
	- [x] [Go汇编](./notes/misc/assembly.md)
	- [x] [runtime hacking](./notes/misc/runtime.md)
	- [x] [GDB使用](./notes/misc/gdb.md)
	- [x] [CPU亲和性](./notes/misc/cpu_affinity.md)
