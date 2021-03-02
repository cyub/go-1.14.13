# Golang 源码分析

Golang版本是go1.14.13

## 目录

- 运行流程
	- [ ] [启动流程](./notes/bootstrap/load.md)
- 堆栈
	- [ ] [Goroutine栈设计](./notes/go-stack.md)
- 错误处理
	- [ ] [panic-defer-recover](./notes/error/panic.md)
- sync
	- [x] [Map](./notes/sync/map.md)
	- [x] [Waitgroup](./notes/sync/waitgroup.md)
- slice
	- [ ] [slice](./notes/slice/slice.md)
- misc
	- [x] [no copy机制](./notes/misc/nocopy.md)
	- [x] [Go汇编](./notes/misc/assembly.md)
	- [x] [runtime hacking](./notes/misc/runtime.md)
	- [ ] [逃逸分析](./notes/misc/escape-analysis.md)
	- [x] [GDB使用](./notes/misc/gdb.md)
	- [ ] [Function Value、闭包、方法](./notes/misc/function_closure_method.md)
	- [x] [值传递、引用传递、值类型变量、引用类型变量](./notes/misc/pass_by_value.md)
