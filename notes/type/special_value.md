# 特殊值与特殊结构

## 零值

[零值(zero value)](https://go.dev/ref/spec#The_zero_value)指的是当声明变量且未显示初始化时，Go语言会自动给变量赋予一个默认初始值。对于值类型变量来说不同值类型，有不同的零值，比如整数型零值是0，字符串类型是""，布尔类型是false。对于引用类型变量其零值都是nil。

类型 | 零值
--- | ---
数值类型 | 0
字符串 | ""
布尔类型| false
指针类型 | nil
通道 | nil
函数 | nil
接口 | nil
映射 | nil
切片 | nil
结构体 | 每个结构体字段对应类型的零值

## nil

`nil`是Go语言中的一个变量，是预先声明的标识符，用来作为引用类型变量的零值。

```go
// nil is a predeclared identifier representing the zero value for a
// pointer, channel, func, interface, map, or slice type.
var nil Type // Type must be a pointer, channel, func, interface, map, or slice type
```

nil不能通过:=方式赋值给一个变量，下面代码是编译不通过的：

```go
a := nil
```

上面代码编译不通过是因为Go语言是无法通过nil自动推断出a的类型，而Go语言是强类型的。下面将nil赋值一个变量是可以的：

```go
var a chan int
a = nil

b := make([]int, 5)
b = nil
```

### 与nil进行比较

#### nil 与 nil比较

nil是不能和nil比较的：

```go
func main() {
	fmt.Println(nil == nil) // 报错：invalid operation: nil == nil (operator == not defined on nil)
}
```

#### nil 与 指针类型变量、通道、切片、函数、映射比较

nil 是可以和指针类型变量，通道、切片、函数、映射比较的。

对于指针类型变量，只有其未指向任何对象时候，才能等于nil：

```go
func main() {
	var p *int
	println(p == nil) // true
	a := 100
	p = &a
	println(p == nil) // false
}
```

对于通道、切片、映射只有`var t T`或者手动赋值为nil时候(`t = nil`)，才能等于nil:

```go
func main() {
	// 通道
	var ch chan int
	println(ch == nil) // true
	ch = make(chan int, 0)
	println(ch == nil) // false

	ch1 := make(chan int, 0)
	println(ch1 == nil) // false
	ch1 = nil
	println(ch1 == nil) // true

	// 切片
	var s []int // 此时s是nil slice
	println(s == nil) // true
	s = make([]int, 0, 0) // 此时s是empty slice
	println(s == nil) // false

	// 映射
	var m map[int]int // 此时m是nil map
	println(m == nil) // true
	m = make(map[int]int, 0)
	println(m == nil) // false

	// 函数
	var fn func()
	println(fn == nil)
	fn = func() {
	}
	println(fn == nil)
}
```

从上面可以看到，通过make函数初始化的变量都不等于nil。

#### nil 与 接口比较

接口类型变量包含两个基础属性：Type和Value，Type指的是接口类型变量的底层类型，Value指的是接口类型变量的底层值。**接口类型变量是可以比较的**。**当它们具有相同的底层类型，且相等的底层值时候，或者两者都为nil时候，这两个接口值是相等的**。

当nil 与接口比较时候，需要接口的Type和Value都是nil时候，才能相等：

```go
func main() {
	var p *int
	var i interface{}                   // (T=nil, V=nil)
	println(p == nil)                   // true
	println(i == nil)                   // true
	var pi interface{} = interface{}(p) // (T=*int, V= nil)
	println(pi == nil)                  // false
	println(pi == i)                    // fasle
	println(p == i)                     // false。跟上面强制转换p一样。当变量和接口比较时候，会隐式将其转换成接口

	var a interface{} = nil // (T=nil, V=nil)
	println(a == nil) // true
	var a2 interface{} = (*interface{})(nil) // (T=*interface{}, V=nil)
	println(a2 == nil) // false
	var a3 interface{} = (interface{})(nil) // (T=nil, V=nil)
	println(a3 == nil) // true
}
```

nil和接口比较最容易出错的场景是使用error接口时候。Go官方文档举了一个例子[Why is my nil error value not equal to nil?](https://golang.org/doc/faq#nil_error):

```go
type MyError int
func (e *MyError) Error() string {
    return "errCode " + string(int)
}

func returnError() error {
	var p *MyError = nil
	if bad() { // 出现错误时候，返回MyError
		p = &MyError(401)
	}
	// println(p == nil) // 输出true
	return p
}

func checkError(err error) {
	if err == nil {
		println("nil")
		return
	}
	println("not nil")
}

err := returnError() // 假定returnsError函数中bad()返回false
println(err == nil) // false
checkError(err) // 输出not nil
```

我们可以看到上面代码中checkError函数输出的并不是nil，而是not nil。这是因为接口类型变量err的底层类型是(T=*MyError, V=nil)，不再是(T=nil, V=nil)。解决办法是当需返回nil时候，直接返回nil

```go
func returnError() error {
	if bad() { // 出现错误时候，返回MyError
		return &MyError(401)
	}
	return p
}
```

### 几个值为nil的特别变量

#### nil通道

通道类型变量的零值是nil，对于等于nil的通道称为nil通道。当从nil通道读取或写入数据时候，会发生永久性阻塞，若关闭则会发生恐慌。nil通道存在的意义可以参考[Why are there nil channels in Go?](https://medium.com/justforfunc/why-are-there-nil-channels-in-go-9877cc0b2308)

#### nil切片

对nil切片进行读写操作时候会发生panic。但对nil切片进行append操作时候是可以的，这是因为Go语言对append操作做了处理。

```go
var s []int
s[0] = 1 // panic: runtime error: index out of range [0] with length 0
println(s[0]) // panic: runtime error: index out of range [0] with length 0
s = append(s, 100) // ok
```

#### nil映射

我们可以对nil映射进行读取和删除操作，当进行读取操作时候会返回映射的零值。当进行写操作时候会发生恐慌。

```go
func main() {
	var m map[int]int
	println(m[100]) // print 0
	delete(m, 1)
	m[100] = 100 // panic: assignment to entry in nil map
}
```

#### nil接收者

值为nil的变量可以作为函数的接收者：

```go
const defaultPath = "/usr/bin/"

type Config struct {
	path string
}

func (c *Config) Path() string {
	if c == nil {
		return defaultPath
	}
	return c.path
}

func main() {
	var c1 *Config
	var c2 = &Config{
		path: "/usr/local/bin/",
	}
	fmt.Println(c1.Path(), c2.Path())
}
```

#### nil函数

nil函数可以用来处理默认值情况：

```go
func NewServer(logger function) {
	if logger == nil {
		logger = log.Printf  // default
	}
	logger.DoSomething...
}
```

## 空结构体 struct{}

空结构体指的是没有任何字段的结构体。

### 大小与内存地址

空结构体占用的内存空间大小为零字节，并且它们的地址可能相等也可能不等。当发生内存逃逸时候，它们的地址是相等的，都指向`runtime.zerobase`。

```go
// empty_struct.go
type Empty struct{}

//go:linkname zerobase runtime.zerobase
var zerobase uintptr // 使用go:linkname编译指令，将zerobase变量指向runtime.zerobase

func main() {
	a := Empty{}
	b := struct{}{}

	fmt.Println(unsafe.Sizeof(a) == 0) // true
	fmt.Println(unsafe.Sizeof(b) == 0) // true
	fmt.Printf("%p\n", &a)             // 0x590d00
	fmt.Printf("%p\n", &b)             // 0x590d00
	fmt.Printf("%p\n", &zerobase)      // 0x590d00

	c := new(Empty)
	d := new(Empty)
	fmt.Sprint(c, d) // 目的是让变量c和d发生逃逸
	println(c) // 0x590d00
	println(d) // 0x590d00
	fmt.Println(c == d) // true

	e := new(Empty)
	f := new(Empty)
	println(e)          // 0xc00008ef47
	println(f)          // 0xc00008ef47
	fmt.Println(e == f) // flase
}
```

从上面代码输出可以看到`a`, `b`, `zerobase`这三个变量的地址都是一样的，最终指向的都是全局变量`runtime.zerobase`([runtime/malloc.go](https://github.com/golang/go/blob/go1.14.13/src/runtime/malloc.go#L827))。

```go
// base address for all 0-byte allocations
var zerobase uintptr
```

我们可以通过下面方法再次来验证一下`runtime.zerobase`变量的地址是不是也是`0x590d00`：

```bash
go build -o empty_struct empty_struct.go
go tool nm ./empty_struct | grep 590d00
# 或者
objdump -t empty_struct | grep 590d00
```

执行上面命令输出以下的内容：

```
590d00 D runtime.zerobase
# 或者
0000000000590d00 g     O .noptrbss	0000000000000008 runtime.zerobase
```

从上面输出的内容可以看到`runtime.zerobase`的地址也是`0x590d00`。


接下来我们看看变量逃逸的情况：

```
 go run -gcflags="-m -l" empty_struct.go
# command-line-arguments
./empty_struct.go:15:2: moved to heap: a
./empty_struct.go:16:2: moved to heap: b
./empty_struct.go:18:13: ... argument does not escape
./empty_struct.go:18:31: unsafe.Sizeof(a) == 0 escapes to heap
./empty_struct.go:19:13: ... argument does not escape
./empty_struct.go:19:31: unsafe.Sizeof(b) == 0 escapes to heap
./empty_struct.go:20:12: ... argument does not escape
./empty_struct.go:21:12: ... argument does not escape
./empty_struct.go:22:12: ... argument does not escape
./empty_struct.go:24:10: new(Empty) escapes to heap
./empty_struct.go:25:10: new(Empty) escapes to heap
./empty_struct.go:26:12: ... argument does not escape
./empty_struct.go:29:13: ... argument does not escape
./empty_struct.go:29:16: c == d escapes to heap
./empty_struct.go:31:10: new(Empty) does not escape
./empty_struct.go:32:10: new(Empty) does not escape
./empty_struct.go:35:13: ... argument does not escape
./empty_struct.go:35:16: e == f escapes to heap
```

可以看到变量`c`和`d`逃逸到堆上，它们打印出来的都是`0x591d00`，且两者进行相等比较时候返回`true`。而变量`e`和`f`打印出来的都是`0xc00008ef47`，但两者进行相等比较时候却返回`false`。这因为Go有意为之的，当空结构体变量未发生逃逸时候，指向该变量的指针是不等的，当空结构体变量发生逃逸之后，指向该变量是相等的。这也就是[Go官方语法指南](https://go.dev/ref/spec)所说的：

> Pointers to distinct zero-size variables may or may not be equal

```eval_rst
.. image:: http://static.cyub.vip/images/202201/go-compare-operators.png
    :alt: Go比较操作符
    :width: 800px
    :align: center
```

### 当一个结构体嵌入空结构体时，占用空间怎么计算？

空结构体本身不占用空间，但是作为某结构体内嵌字段时候，有可能是占用空间的：

- 当空结构体是该结构体唯一的字段时，该结构体是不占用空间的，空结构体自然也不占用空间
- 当空结构体作为第一个字段或者中间字段时候，是不占用空间的
- 当空结构体作为最后一个字段时候，是占用空间的，大小跟其前一个字段保持一致

```go
type s1 struct {
	a struct{}
}

type s2 struct {
	_ struct{}
}

type s3 struct {
	a struct{}
	b byte
}

type s4 struct {
	a struct{}
	b int64
}

type s5 struct {
	a byte
	b struct{}
	c int64
}

type s6 struct {
	a byte
	b struct{}
}

type s7 struct {
	a int64
	b struct{}
}

type s8 struct {
	a struct{}
	b struct{}
}

func main() {
	fmt.Println(unsafe.Sizeof(s1{})) // 0
	fmt.Println(unsafe.Sizeof(s2{})) // 0
	fmt.Println(unsafe.Sizeof(s3{})) // 1
	fmt.Println(unsafe.Sizeof(s4{})) // 8
	fmt.Println(unsafe.Sizeof(s5{})) // 16
	fmt.Println(unsafe.Sizeof(s6{})) // 2
	fmt.Println(unsafe.Sizeof(s7{})) // 16
	fmt.Println(unsafe.Sizeof(s8{})) // 0
}
```

当空结构体作为数组、切片的元素时候：

```go
var a [10]int
fmt.Println(unsafe.Sizeof(a)) // 80

var b [10]struct{}
fmt.Println(unsafe.Sizeof(b)) // 0

var c = make([]struct{}, 10)
fmt.Println(unsafe.Sizeof(c)) // 24，即slice header的大小
```

### 用途

由于空结构体占用的空间大小为零，我们可以利用这个特性，完成一些功能，却不需要占用额外空间。

### 阻止`unkeyed`方式初始化结构体

```go
type MustKeydStruct struct {
	Name string
	Age  int
	_    struct{}
}

func main() {
	persion := MustKeydStruct{Name: "hello", Age: 10}
	fmt.Println(persion)
	persion2 := MustKeydStruct{"hello", 10} //编译失败，提示： too few values in MustKeydStruct{...}
	fmt.Println(persion2)
}
```

#### 实现集合数据结构

集合数据结构我们可以使用map来实现：只关心key，不必关心value，我们就可以值设置为空结构体类型变量（或者底层类型是空结构体的变量）。

```go
package main

import (
	"fmt"
)

type Set struct {
	items map[interface{}]emptyItem
}

type emptyItem struct{}

var itemExists = emptyItem{}

func NewSet() *Set {
	set := &Set{items: make(map[interface{}]emptyItem)}
	return set
}

// 添加元素到集合
func (set *Set) Add(item interface{}) {
	set.items[item] = itemExists
}

// 从集合中删除元素
func (set *Set) Remove(item interface{}) {
	delete(set.items, item)

}

// 判断元素是否存在集合中
func (set *Set) Contains(item interface{}) bool {
	_, contains := set.items[item]
	return contains
}

// 返回集合大小
func (set *Set) Size() int {
	return len(set.items)
}

func main() {
	set := NewSet()
	set.Add("hello")
	set.Add("world")
	fmt.Println(set.Contains("hello"))
	fmt.Println(set.Contains("Hello"))
	fmt.Println(set.Size())
}
```

#### 作为通道的信号传输

使用通道时候，有时候我们只关心是否有数据从通道内传输出来，而不关心数据内容，这时候通道数据相当于一个信号，比如我们实现退出时候。下面例子是基于通道实现的信号量。

```go
// empty struct
var empty = struct{}{}

// Semaphore is empty type chan
type Semaphore chan struct{}

// P used to acquire n resources
func (s Semaphore) P(n int) {
	for i := 0; i < n; i++ {
		s <- empty
	}
}

// V used to release n resouces
func (s Semaphore) V(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

// Lock used to lock resource
func (s Semaphore) Lock() {
	s.P(1)
}

// Unlock used to unlock resource
func (s Semaphore) Unlock() {
	s.V(1)
}

// NewSemaphore return semaphore
func NewSemaphore(N int) Semaphore {
	return make(Semaphore, N)
}
```

## 进一步阅读

- [Golang 零值、空值与空结构](https://juejin.cn/post/6895231755091968013)
- [Why are there nil channels in Go?](https://medium.com/justforfunc/why-are-there-nil-channels-in-go-9877cc0b2308)
- [The empty struct](https://dave.cheney.net/2014/03/25/the-empty-struct)