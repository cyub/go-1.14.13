## Function value

Golang中函数是一等公民，函数可以绑定到变量，也可以做参数传递以及做函数返回值，Golang把这样的参数、返回值、变量称为`function value`。

`function value`本质上是一个指针，但是并不直接指向函数入口地址。而是直接指向[`runtime.funcval`](https://github.com/cyub/go-1.14.13/blob/master/src/runtime/runtime2.go#L195)结构体。这个结构体中的fn存储的就是闭包函数的入口地址。

```go
type funcval struct {
	fn uintptr
}
```

我们以下面这段代码为例来看下`function value`是如何使用的:

```go
func A(i int) {
	i++
	fmt.Println(i)
}

func B() {
	f1 := A
	f1(1)
}

func C() {
	f2 := A
	f2(2)
}
```

上面代码中，函数A被赋值给变量f1和f2，这种情况下编译器会做出优化，让f1和f2共用一个`funcval`结构体，该结构体是在编译阶段分配到数据段的只读区域(.rodata)。如下图所示那样，f1和f2都指向了该结构体的地址`addr2`，该结构体的fn字段存储了函数A的入口地址`addr1`：

![](https://static.cyub.vip/images/202102/go_function_value.jpeg)


为什么f1和f2需要通过了一个二级指针来获取到真正的函数入口地址，而不是直接将f1，f2指向函数入口地址`addr1`。关于这个原因就涉及到Golang中闭包设计与实现了。

## Closure

**闭包(Closure)通俗点讲就是能够访问外部函数内部变量的函数**。像这样能被访问的变量通常被称为**捕获变量**。

闭包函数指令在编译阶段生成，但因为每个闭包对象都要保存自己捕获的变量，所以要等到执行阶段才创建对应的闭包对象。我们来看下下面闭包的例子：

```go
func create() func() int {
	c := 2
	return func() int { // 闭包函数
		return c
	}
}

func main() {
	f1 := create()
	f2 := create()

	print(f1())
	print(f2())
}
```

上面代码中当执行main函数时，会在其栈帧区间内为局部变量f1和f2分配栈空间，当执行第一个create函数时候，会在其栈帧空间分配栈空间来存放局部变量c，然后在**堆**上分配一个`funcval`结构体（其地址假定addr2)，该结构体的fn字段存储的是create函数内那个闭包函数的入口地址（其地址假定为addr1）。create函数除了分配一个`funcval`结构体外，还会挨着该结构体分配闭包函数的变量捕获列表，该捕获列表里面只有一个变量c。由于**捕获列表的存在，所以说闭包函数是一个有状态函数**。

当create函数执行完毕后，返回值赋值给f1，此时f1指向的就是地址addr2。同理下来f2指向地址addr3。f1和f2都能通过`funcval`取到了闭包函数入口地址，但拥有不同的捕获列表。

当执行f1()时候，Golang会将其对应`funcval`地址存储到特定寄存器（比如amd64平台中使用rax寄存器），这样在闭包函数中就可以通过该寄存器取出`funcval`地址，然后通过偏移找到每一个捕获的变量。由此可以看出来**Golang中闭包就是有捕获列表的Function value**。

内存分布示例图如下：

![](https://static.cyub.vip/images/202102/go_closure_function.jpeg)

### 捕获变量更改情况

上面例子中被捕获的变量没有被更改过，所以Golang很智能地只是将create中局部变量c拷贝到f1和f2的捕获列表中。

对于捕获变量值会改变的闭包函数，Golang中又是怎么做到捕获变量在外层函数和闭包函数保持一致的，好像在使用同一个变量？让我们来看看下面这个例子：

```go
func create() (fs [2]func()) {
	for i:=0; i<2; i++ {
		fs[i] = func() {
			print(i)
		}
	}
	return
}

func main() {
	fs := create()
	for i:=0; i< len(fs); i++ {
		fs[i]()
	}
}
```

上面代码中create函数会创建两个闭包函数，并且闭包函数会修改闭包变量。当执行main函数时，创建局部变量fs，其是长度为2的`function value`类型数组。当执行到create函数时候，由于局部变量i会被闭包函数捕获，且会被修改，变量i会发生内存逃逸改成堆分配，并在create栈上存储其在堆中的地址，此外fs[0]和fs[1]的捕获列表中存储的也是这个堆地址。这样create函数以及闭包函数fs[0]、fs[1]访问都是堆上的同一个变量。这样三者使用是同一个变量，这也是为什么fs[0],fs[1]最后打印出来都是2的原因。

由上面例子我们也可以发现**闭包导致的局部变量堆分配也是内存逃逸的一种情况**。上面例子是**捕获并修改的是外层函数局部变量的情况**，除此之外还有以下两种情况也会发生内存逃逸。

**当捕获并修改的是外层函数参数的时候**，Go会将该外层函数的调用者栈上的参数拷贝到堆上(函数的参数是由其调用者分配空间的），然后外层函数和闭包函数都是用堆上分配的参数。

**当捕获的是外层函数返回值的时候**，闭包的调用者函数会在堆上分配返回值空间，然后外层函数和闭包函数都使用堆上这个返回值空间，在外层函数返回之前，会将堆上的返回值拷贝到该外层函数调用者为其分配的返回值空间中。

## Method

方法指的是一段被它关联的对象通过它的名字调用的代码块。比如下面的golang方法代码：

```go
type A struct {
    name string
}

func (a A) Name() string {
    a.name = "Hi " + a.name
    return a.name
}

func main() {
    a := A{name: "new world"}
    fmt.Println(a.Name())
    fmt.Println(A.Name(a))
}

func NameName(a A) string {
    a.name = "Hi " + a.name
    return a
}
```
上面代码中`a.Name()`代表的函数是调用a对象的Name方法。它实际上是个语法糖，等效于`A.Name(a)`，a是方法接收者，它会做方法Name的第一个参数传入。我们可以通过以下代码证明两者是相等的：

```go
t1 := reflect.TypeOf(A.Name)
t2 := relect.TypeOf(NameOfA)

fmt.Println(t1 == t2) // true
```

所以说**方法本质就是普通的函数，接收者（其他语言就是类对象了）就是隐含的第一个参数**。

我们来看下值接收者和指针接收者方法混合的情况：

```go
type A struct {
    name string
}

func (a A) GetName() string {
    return a.name
}

func (pa *A) SetName() string {
    pa.name = "Hi " + p.name
    return pa.name
}

func main() {
    a := A{name: "new world"}
    pa := &a

    fmt.Println(pa.GetName())
    fmt.Println(a.SetName())
}
```

上面代码中通过指针调用值接收者方法和通过值调用指针接收者方法，都能够正常运行，因为两者都是语法糖，golang在编译阶段会将两者转换如下形式：

```go
fmt.Println((*pa).GetName())
fmt.Println((&a).SetName())
```

由于是编译阶段实现的语法糖，所以对于编译期间拿不到地址的字面量（比如(A{name:"hi"}).SetName())就不能通过编译运行了。

### 方法表达式与方法变量

```go
type A struct {
    name string
}

func (a A) GetName() string {
    return a.name
}

func main() {
    a := A{name: "new world"}

    f1 := A.GetName // 方法表达式
    f1(a)

    f2 := a.GetName // 方法变量
    f2()
}
```

方法表达式(Method Expression) 与方法变量(Method Value)本质上将都是`Function Value`，区别在于方法变量会捕获方法接收者形成闭包，此方法变量的生命周期与方法接收一样，编译会将其进行优化转换成对类型T的方法调用，并传入接收者作为参数。

根据上面描述我们可以将上面代码中`f2`理解成如下代码：

```go
func GetFunc() (func()) string {
    a := A{name: "new world"}
    return func() string {
        return A.GetName(a)
    }
}

f2 = GetFunc()
```