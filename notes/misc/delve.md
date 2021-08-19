# delve

delve是专门用来调试Go应用的利器。它和GDB类似，支持打断点，单步调试，Examin内存等功能。delve跟踪调试应用程序的原理和GDB实现原理类似，都是使用ptrace这个系统调用。

delve启动调试时候，会调用ptrace系统调用，将待调试跟踪的Go应用程序的pid传递给ptrace后，delve会成该Go应用执行的`tracer`（跟踪者），该Go应用称为`tracee`（被跟踪者），其会被标记为`traced`(跟踪）状态。此后delve可以查看`tracee`的内存和寄存器。

在`tracee`在执行系统调用之前，系统内核会先检查`tracee`是否处于被`traced`的状态。如果是的话，内核暂停当前`tracee`执行并将控制权交给`tracer`，使`tracer`得以查看或者修改`tracee`的寄存器或内存信息。

## delve运行架构图

![](https://static.cyub.vip/images/202107/delve.png)



