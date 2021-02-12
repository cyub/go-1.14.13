## 简介

Go中一些数据结构是`no copy`的。这里面的`no copy`指的是第一次使用之后，该变量不能赋值给另外一个变量来使用，或者通过函数值传递来使用。对于这些数据结构，Go源码中一般都会添加`must not be copied after first use`注释来说明：

```bash
➜  go-1.14.13 git:(master) grep -r 'must not be copied after first use' ./src
./src/cmd/go/internal/lockedfile/mutex.go:// must not be copied after first use. The Path field must be set before first
./src/sync/atomic/value.go:// A Value must not be copied after first use.
./src/sync/cond.go:// A Cond must not be copied after first use.
./src/sync/rwmutex.go:// A RWMutex must not be copied after first use.
./src/sync/map.go:// The zero Map is empty and ready for use. A Map must not be copied after first use.
./src/sync/mutex.go:// A Mutex must not be copied after first use.
./src/sync/pool.go:// A Pool must not be copied after first use.
./src/sync/waitgroup.go:// A WaitGroup must not be copied after first use.
```

对于`no copy`的结构体不能copy的原因是该结构体的某些字段记录了该结构体的状态信息，或者计数器信息等，该结构体提供的方法会更新这些信息，而这些字段是值引用的，当copy之后，调用新变量的方法不会再更新原来结构体对应的信息。

## Go中是如何保证`no copy`的?

Go中提供两种方法来保证`no copy`机制：`runtime checking` 和 `go vet checking`

### runtime checking

`runtime checking`指的在程序运行时检查对象是否进行copy了。Go中对`strings.Builder`和`sync.Cond`提供了runtime checking。

strings.Builder copy示例：
```go
func main() {
	var s1 strings.Builder
	s1.Write([]byte("a"))
	s2 := s1
	s2.Write([]byte("b"))
}
```

strings.Builder中`no copy` [runtime检查源码](https://github.com/cyub/go-1.14.13/blob/master/src/strings/builder.go#L33)：
```go
type Builder struct {
	addr *Builder // of receiver, to detect copies by value
	buf  []byte
}

func (b *Builder) Write(p []byte) (int, error) {
	b.copyCheck()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

func (b *Builder) copyCheck() {
	if b.addr == nil {
		b.addr = (*Builder)(noescape(unsafe.Pointer(b)))
	} else if b.addr != b {
		panic("strings: illegal use of non-zero Builder copied by value")
	}
}
```

string.Builder进行Write时候，会先进行copyCheck。示例中当s1赋值给s2时候，s2的addr指向的还是s1。s2进行Write时候，copyCheck中会触发逻辑`b.addr != b`，发生恐慌。

### go vet checking

`go vet`命令提供了静态检查功能。用来编译时候提供检查。只要实现了`sync.Locker`接口的对象，那么`go vet`就会检查代码中有没有copy这个对象的逻辑。

相比`runtime checking`，`go vet checking`不会影响程序性能。

`sync.Locker`的[定义](https://github.com/cyub/go-1.14.13/blob/master/src/sync/mutex.go#L31)：
```go
type Locker interface {
	Lock()
	Unlock()
}
```

sync包提供了[noCopy类型](https://github.com/cyub/go-1.14.13/blob/master/src/sync/cond.go#L94)，它实现了`sync.Locker`接口
```go
type noCopy struct{}
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
```


`sync.Pool`, `sync.WaitGroup`, `sync.Cond`通过内嵌`noCopy`类型的字段实现`sync.Locker`接口。

```go
type Pool struct {
    noCopy noCopy
    ...
}

type WaitGroup struct {
    noCopy noCopy
    ...
}

type Cond struct {
    noCopy noCopy
    ...
}
```

`sync.Mutex`, `sync.RWMutex`通过自己实现`Lock()`和`Unlock()`方法，实现了`sync.Locker`接口

`sync.Map`通过内嵌`sync.Mutex`来实现`sync.Locker`接口

## 总结

- Go中采用`runtime checking` 和 `go vet checking`两种方法来保证`no copy`机制
- Go中`stings.Builder`, `sync.Pool`, `sync.WaitGroup`, `sync.Cond`, `sync.Map`, `sync.Mutex`, `sync.RWMutext`是`no copy`的


