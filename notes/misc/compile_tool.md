# go tool compile

`go tool compile`称为编译命令，用来生成对象文件，生成对象文件最后通过`go tool link`命令链接成可执行文件。

编译命名用法示例如下：

```
go tool compile [flags] file...
```

在使用编译命令时候，我们可以设置`GOOS` 和 `GOARCH`环境变量来指定目标操作系统和系统架构。这个和`go build`命令是一致的。

`go tool compile`支持的选项有：

选项 | 说明
--- | ---
-D path | Set relative path for local imports.
-I dir1 -I dir2 | Search for imported packages in dir1, dir2, etc,<br/>after consulting $GOROOT/pkg/$GOOS_$GOARCH.
-L | Show complete file path in error messages.
<b>-N</b> | Disable optimizations.
<b>-S</b> | Print assembly listing to standard output (code only).
-S -S | Print assembly listing to standard output (code and data).
-V | Print compiler version and exit.
-asmhdr file | Write assembly header to file.
-asan | Insert calls to C/C++ address sanitizer.
-buildid id | Record id as the build id in the export metadata.
-blockprofile file | Write block profile for the compilation to file.
-c int | Concurrency during compilation. Set 1 for no concurrency (default is 1).
-complete | Assume package has no non-Go components.
-cpuprofile file | Write a CPU profile for the compilation to file.
-dynlink | Allow references to Go symbols in shared libraries (experimental).
-e | Remove the limit on the number of errors reported (default limit is 10).
-goversion string | Specify required go tool version of the runtime. <br/>Exits when the runtime go version does not match goversion.
-h | Halt with a stack trace at the first error detected.
-importcfg file | Read import configuration from file. <br/>In the file, set importmap, packagefile to specify import resolution.
-installsuffix suffix | Look for packages in $GOROOT/pkg/$GOOS_$GOARCH_suffix<br/>instead of $GOROOT/pkg/$GOOS_$GOARCH.
<b>-l</b> | Disable inlining.
-lang version | Set language version to compile, as in -lang=go1.12.<br/>Default is current version.
-linkobj file | Write linker-specific object to file and compiler-specific<br/>object to usual output file (as specified by -o).<br/>Without this flag, the -o output is a combination of both<br/>linker and compiler input.
-m | Print optimization decisions. Higher values or repetition<br/>produce more detail.
-memprofile file | Write memory profile for the compilation to file.
-memprofilerate rate | Set runtime.MemProfileRate for the compilation to rate.
-msan | Insert calls to C/C++ memory sanitizer.
-mutexprofile file | Write mutex profile for the compilation to file.
-nolocalimports | Disallow local (relative) imports.
-o file | Write object to file (default file.o or, with -pack, file.a).
-p path | Set expected package import path for the code being compiled,<br/>and diagnose imports that would cause a circular dependency.
-pack | Write a package (archive) file rather than an object file
-race | Compile with race detector enabled.
-s | Warn about composite literals that can be simplified.
-shared | Generate code that can be linked into a shared library.
-spectre list | Enable spectre mitigations in list (all, index, ret).
-traceprofile file | Write an execution trace to file.
-trimpath prefix | Remove prefix from recorded source file paths.

## 编译指令(Compiler Directives)

Go 编译器可以读取来自注释里面的指令，这些指令称为编译指令，这些编译指令往往以`//go:`开头。

### //line指令

由于历史缘故，line指令并不是由`//go:`开头。它通常出现在机器码生成时候，这样编译器或者调试器才能报告出原始输入文件的位置。比如查看cgo编译过程中的中间文件会出现(go tool cgo xxx.go)。

### //go:noescape

noescape = no + escape。noescape意味着禁止逃逸，该编译指令后面必须跟着函数声明(没有函数体)。该函数的具体实现是由汇编实现的。

### //go:uintptrescapes 

该指令后面必须跟一个函数声明。它表明该函数的 uintptr 参数是一个指针指，需要将其分配到堆上，来保证调用期间不被回收，造成悬挂现象。

### //go:noinline

//go:noinline 指令后面必须跟一个函数定义。它指定不对该函数进行内联优化。这通常仅在特殊运行时函数或调试编译器时需要。

```go
//go:noinline
func add(a, b int)  int {
    return a + b
}

func main() {
    c := add(3, 5)
    fmt.Println(c)
}
```

内联优化的优点：
- 减少函数调用的开销，提高执行速度，毕竟call指令是比较耗时的。
- 消除分支，并改善空间局部性和指令顺序性，同样可以提高性能

内联优化并不是银弹，如果有大量重复代码，反而会降低CPU缓存命中率。需要注意的是Go编译器没有强制使用内联优化的编译指令。


### //go:norace

该指令后面必须跟一个函数声明。它指定函数的内存访问必须被竞争检测器忽略。这最常用于在调用竞争检测器运行时不安全时调用的低级代码。

### //go:nosplit

该指令后面必须跟一个函数声明。它指定函数必须省略其通常的堆栈溢出检查。当调用 goroutine 被抢占是不安全的时候，这最常被调用的低级运行时代码使用。

Go中gorountine的栈初始大小为2K，正常情况下每个函数最开始部分都有一个堆栈检测的序言指令，执行函数前检查goroune栈是否需要扩容。

### //go:linkname localname [importpath.name]

该指令用来将源代码localname符号链接到源代码中的importpath.name符号。由于该指令破坏了类型系统和软件包的模块化，因此使用的时候，需要导入`unsafe`包。

```go
import "unsafe"

//go:linkname zerobase runtime.zerobase
var zerobase uintptr // 使用go:linkname编译指令，将zerobase变量指向runtime.zerobase
```

### //go:build

该指令是用于构建约束(build constraint)，它是在Go1.17版本中引入，之前版本都是通过`+build`来实现的。Go1.17中，为了兼容性，两者都需要保留，否则编译不通过。

```go
//go:build linux && 386
// +build linux,386

func add(a, b int)  int {
    return a + b
}
```

#### 运行时源码中的编译指令

见[Runtime-only 编译指令](https://github.com/cyub/go-1.14.13/blob/6f2baf2ab6a52611e29e94060230e9629c3181f0/notes/misc/runtime.md#runtime-only-%E7%BC%96%E8%AF%91%E6%8C%87%E4%BB%A4compiler-directives)


## 资料

- [Command compile](https://golang.google.cn/pkg/cmd/compile/)
- [Go’s hidden #pragmas](https://dave.cheney.net/2018/01/08/gos-hidden-pragmas)
- [Bug-resistant build constraints — Draft Design](https://go.googlesource.com/proposal/+/master/design/draft-gobuild.md)

