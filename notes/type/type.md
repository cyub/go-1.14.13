## 类型系统

### 为什么需要类型系统？

```go
type T struct {
    name string
}

func (t T) func1() {
    fmt.Println(t.name)
}

func main() {
    t := T{name:"go"}
    t.func1()
}
```

我们知道方法本质上就是函数，当调用方法时候，接收者会作为函数的第一个参数传递进来。这个是在编译阶段完成的。但是当需要动态执行的时候，比如反射、接口、类型断言，就需要数据对象的类型了。


### 内置类型与自定义类型

Golang中类型分为内置类型和自定义类型。其中内置类型是无法自定义方法的，接口类型是无效的接收者类型。

内置类型示例：

```go
# built-in
int8
int16
int32
int64
int
byte
string
map
slice
```

自定义类型示例：

```go
type T int
type T struct {
    name string
}

type I interface{
    Name() string
}
```

不论内置类型，还是自定义类型都会有相对应的类型描述信息，称为其"类型元数据"。每一种类型的元数据信息都是全局唯一的。这些类型元素构成了Go语言的”类型系统“。


### 类型元数据底层结构

Golang中类型元数据信息都存放在[`runtime._type`](https://github.com/cyub/go-1.14.13/blob/master/src/runtime/type.go#L31-L48)这个结构体中，作为每个类型元素的Header。

```go
type _type struct {
	size       uintptr // 类型大小
	ptrdata    uintptr // 指针类型截止处字节大小
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      tflag   // extra type information flags
	align      uint8   // 对齐边界
	fieldAlign uint8   // 对齐边界
	kind       uint8   // 类型
	equal     func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata    *byte   // 用于GC的信息，以位图形式记录信息，当位置为1，说明相应字段是指针类型。
	str       nameOff // string form
	ptrToThis typeOff // type for pointer to this type, may be zero
}
```

在_type之后存储的是各种类型额外需要描述的信息。比如slice的类型元数据在_type结构体之后记录了`*_type`类型信息elem指向了其存储的元素的类型元数据。

```go
type slicetype struct {
	typ  _type
	elem *_type
}
```

其他内置类型的类型元信息：

```go
type arraytype struct {
	typ   _type
	elem  *_type
	slice *_type
	len   uintptr
}

type chantype struct {
	typ  _type
	elem *_type
	dir  uintptr
}

type slicetype struct {
	typ  _type
	elem *_type
}

type functype struct {
	typ      _type
	inCount  uint16
	outCount uint16
}

type ptrtype struct {
	typ  _type
	elem *_type
}

type structfield struct {
	name       name
	typ        *_type
	offsetAnon uintptr
}

func (f *structfield) offset() uintptr {
	return f.offsetAnon >> 1
}

type structtype struct {
	typ     _type
	pkgPath name
	fields  []structfield
}
```

对于其他自定义类型的元数据的在其他描述信息后面还有`uncommontype`结构体信息：

```go
type uncommontype struct {
	pkgpath nameOff // 记录类型所在的包路径
	mcount  uint16 // 方法数量
	xcount  uint16 // exported方法数量
	moff    uint32 // [mcount]method数组相对于该uncommontype偏移的字节数
	_       uint32 // unused
}

type method struct { // 方法信息
	name nameOff
	mtyp typeOff
	ifn  textOff
	tfn  textOff
}
```

我们现在考虑下面类型的类型元数据信息：

```go
type myslice []string

func (ms myslice) Len() {
    fmt.Println(len(ms))
}

func (ms myslice) Cap() {
    fmt.Println(cap(ms))
}
```

myslice是基于slice的自定义类型：

![](https://static.cyub.vip/images/202103/myslice_type.png)


### 类型别名与类型定义

```go
type MyType1 = int32

type MyType2 = int32
```

上面MyType1是int32的别名，实际上Mytype1和int32会关联到同一个类型元数据，属于同一种类型。Mytype2是基于已有类型创建了新类型，Mytype2拥有了和int32不一样的类型元数据，即使它内容没有更改。