## slice

### 源码分析

让我们从汇编的视角来分析slice的定义与扩容的过程。

```go
package main

import "fmt"

func main() {
	a := []int{1, 3}
	a = append(a, 5, 6, 7)
	fmt.Println(len(a), cap(a))
}
```

根据上面go代码生成汇编代码：

```
go tool compile -N -S -l main.go > main.s
```

main.s文件主要内容如下，或者[在线查看](https://go.godbolt.org/z/PcT9qh)：

```as
"".main STEXT size=497 args=0x0 locals=0xd8
	0x0000 00000 (main.go:5)	TEXT	"".main(SB), ABIInternal, $216-0
	0x0000 00000 (main.go:5)	MOVQ	(TLS), CX
	0x0009 00009 (main.go:5)	LEAQ	-88(SP), AX
	0x000e 00014 (main.go:5)	CMPQ	AX, 16(CX)
	0x0012 00018 (main.go:5)	PCDATA	$0, $-2
	0x0012 00018 (main.go:5)	JLS	487
	0x0018 00024 (main.go:5)	PCDATA	$0, $-1
	0x0018 00024 (main.go:5)	SUBQ	$216, SP
	0x001f 00031 (main.go:5)	MOVQ	BP, 208(SP)
	0x0027 00039 (main.go:5)	LEAQ	208(SP), BP
	0x002f 00047 (main.go:5)	PCDATA	$0, $-2
	0x002f 00047 (main.go:5)	PCDATA	$1, $-2
	0x002f 00047 (main.go:5)	FUNCDATA	$0, gclocals·0ce64bbc7cfa5ef04d41c861de81a3d7(SB)
	0x002f 00047 (main.go:5)	FUNCDATA	$1, gclocals·20f1609b7a8896a9b6839558f8debf33(SB)
	0x002f 00047 (main.go:5)	FUNCDATA	$2, gclocals·f6aec3988379d2bd21c69c093370a150(SB)
	0x002f 00047 (main.go:5)	FUNCDATA	$3, "".main.stkobj(SB)
	0x002f 00047 (main.go:6)	PCDATA	$0, $0
	0x002f 00047 (main.go:6)	PCDATA	$1, $0
	0x002f 00047 (main.go:6)	XORPS	X0, X0
	0x0032 00050 (main.go:6)	MOVUPS	X0, ""..autotmp_6+80(SP)
	0x0037 00055 (main.go:6)	PCDATA	$0, $1
	0x0037 00055 (main.go:6)	LEAQ	""..autotmp_6+80(SP), AX
	0x003c 00060 (main.go:6)	MOVQ	AX, ""..autotmp_4+112(SP)
	0x0041 00065 (main.go:6)	TESTB	AL, (AX)
	0x0043 00067 (main.go:6)	MOVQ	$1, ""..autotmp_6+80(SP)
	0x004c 00076 (main.go:6)	TESTB	AL, (AX)
	0x004e 00078 (main.go:6)	MOVQ	$3, ""..autotmp_6+88(SP)
	0x0057 00087 (main.go:6)	TESTB	AL, (AX)
	0x0059 00089 (main.go:6)	JMP	91
	0x005b 00091 (main.go:6)	MOVQ	AX, "".a+128(SP)
	0x0063 00099 (main.go:6)	MOVQ	$2, "".a+136(SP)
	0x006f 00111 (main.go:6)	MOVQ	$2, "".a+144(SP)
	0x007b 00123 (main.go:7)	JMP	125
	0x007d 00125 (main.go:7)	PCDATA	$0, $2
	0x007d 00125 (main.go:7)	LEAQ	type.int(SB), CX
	0x0084 00132 (main.go:7)	PCDATA	$0, $1
	0x0084 00132 (main.go:7)	MOVQ	CX, (SP)
	0x0088 00136 (main.go:7)	PCDATA	$0, $0
	0x0088 00136 (main.go:7)	MOVQ	AX, 8(SP)
	0x008d 00141 (main.go:7)	MOVQ	$2, 16(SP)
	0x0096 00150 (main.go:7)	MOVQ	$2, 24(SP)
	0x009f 00159 (main.go:7)	MOVQ	$5, 32(SP)
	0x00a8 00168 (main.go:7)	CALL	runtime.growslice(SB)
	0x00ad 00173 (main.go:7)	PCDATA	$0, $1
	0x00ad 00173 (main.go:7)	MOVQ	40(SP), AX
	0x00b2 00178 (main.go:7)	MOVQ	48(SP), CX
	0x00b7 00183 (main.go:7)	MOVQ	56(SP), DX
	0x00bc 00188 (main.go:7)	ADDQ	$3, CX
	0x00c0 00192 (main.go:7)	JMP	194
	0x00c2 00194 (main.go:7)	MOVQ	$5, 16(AX)
	0x00ca 00202 (main.go:7)	MOVQ	$6, 24(AX)
	0x00d2 00210 (main.go:7)	MOVQ	$7, 32(AX)
	0x00da 00218 (main.go:7)	PCDATA	$0, $0
	0x00da 00218 (main.go:7)	PCDATA	$1, $1
	0x00da 00218 (main.go:7)	MOVQ	AX, "".a+128(SP)
	0x00e2 00226 (main.go:7)	MOVQ	CX, "".a+136(SP)
	0x00ea 00234 (main.go:7)	MOVQ	DX, "".a+144(SP)
	0x00f2 00242 (main.go:8)	MOVQ	CX, ""..autotmp_2+72(SP)
	0x00f7 00247 (main.go:8)	PCDATA	$1, $0
	0x00f7 00247 (main.go:8)	MOVQ	"".a+144(SP), AX
```

内容有点多，我们分段来看，先看main函数栈空间分配部分：

```as
"".main STEXT size=497 args=0x0 locals=0xd8
	0x0000 00000 (main.go:5)	TEXT	"".main(SB), ABIInternal, $216-0 // 216是main函数的栈帧大小， 0是参数以及返回值的小
	0x0000 00000 (main.go:5)	MOVQ	(TLS), CX // 本地线程存储，不用管
	0x0009 00009 (main.go:5)	LEAQ	-88(SP), AX
	0x000e 00014 (main.go:5)	CMPQ	AX, 16(CX)
	0x0012 00018 (main.go:5)	PCDATA	$0, $-2
	0x0012 00018 (main.go:5)	JLS	487 // 栈自动伸缩处理的逻辑，不用管
	0x0018 00024 (main.go:5)	PCDATA	$0, $-1
	0x0018 00024 (main.go:5)	SUBQ	$216, SP // 分配216字节栈空间
	0x001f 00031 (main.go:5)	MOVQ	BP, 208(SP) //  把调用函数基址入栈
	0x0027 00039 (main.go:5)	LEAQ	208(SP), BP // 把被调用函数即main函数的基址存入BP寄存器
```

上面汇编的主要逻辑是给main函数分配栈帧空间。Go汇编采用的plan 9汇编，其函数调用协议采用`caller-save`模式，调用函数（caller func)来管理被调用函数(callee func)的参数和返回值。main函数分配的216字节空间有部分用于被调用函数的参数和返回值存储。

接下来看切片a定义时候的汇编代码，其对应的go代码：

```go
a := []int{1, 3}
```

相应的汇编代码如下：

```as
0x0037 00055 (main.go:6)	LEAQ	""..autotmp_6+80(SP), AX // 把80(SP) 地址加载到AX寄存器， AX寄存器存的是切片a底层数组的地址
0x003c 00060 (main.go:6)	MOVQ	AX, ""..autotmp_4+112(SP) // 把AX中值加载到120(SP)
0x0041 00065 (main.go:6)	TESTB	AL, (AX) // 无关指令，不用关心
0x0043 00067 (main.go:6)	MOVQ	$1, ""..autotmp_6+80(SP) // 切片a第一个元素1加载到数组中
0x004c 00076 (main.go:6)	TESTB	AL, (AX)
0x004e 00078 (main.go:6)	MOVQ	$3, ""..autotmp_6+88(SP) // 切片a第二个元素3加载到数组中
0x0057 00087 (main.go:6)	TESTB	AL, (AX)
0x0059 00089 (main.go:6)	JMP	91
0x005b 00091 (main.go:6)	MOVQ	AX, "".a+128(SP) // 切片a的底层数组地址: array
0x0063 00099 (main.go:6)	MOVQ	$2, "".a+136(SP) // 切片a的长度: len
0x006f 00111 (main.go:6)	MOVQ	$2, "".a+144(SP) // 切片a的容器: cap
```

从Go源码中可以看到切片底层数据结构是一个结构体：
```go
type slice struct {
	array unsafe.Pointer // array是指针，指向一个数组，切片元素都存在该数组中
	len   int // 切片长度
	cap   int // 切片容量
}
```


结合汇编代码和切片底层结构，画出切片a在内存中存储结构图：

![](https://static.cyub.vip/images/202102/slice_define_assembly.jpeg)

注意入口函数main是由proc.go中main函数调用起来的。

接下来看下对切片a进行append操作，这部分涉及到切片扩容操作：

对切片进行append操作时候，会先调用`runtime.growslice`函数对slice进行扩容，该返回一个新的slice底层结构体，该结构体array字段指向新的底层数组地址，cap字段是新切片的容量，**len字段是旧切片的长度**。

```as
0x007d 00125 (main.go:7)	LEAQ	type.int(SB), CX // 把int类型定义加载到CX中
0x0084 00132 (main.go:7)	PCDATA	$0, $1
0x0084 00132 (main.go:7)	MOVQ	CX, (SP) // CX加载到(SP)，即(SP) = type.int(SB)，(SP)存储的是切片a的元素类型
0x0088 00136 (main.go:7)	PCDATA	$0, $0
0x0088 00136 (main.go:7)	MOVQ	AX, 8(SP) // 切片a的底层数组地址
0x008d 00141 (main.go:7)	MOVQ	$2, 16(SP) // 切片a的len
0x0096 00150 (main.go:7)	MOVQ	$2, 24(SP) // 切片a的cap
0x009f 00159 (main.go:7)	MOVQ	$5, 32(SP) // 新容量大小
0x00a8 00168 (main.go:7)	CALL	runtime.growslice(SB)
0x00ad 00173 (main.go:7)	PCDATA	$0, $1
0x00ad 00173 (main.go:7)	MOVQ	40(SP), AX // 40(SP)存储的是growslice函数返回的底层数组地址， 即append返回的新切片的底层数组地址
0x00b2 00178 (main.go:7)	MOVQ	48(SP), CX // 旧切片的长度，注意growslice返回的不是新切片的长度，而是旧的切片长度
0x00b7 00183 (main.go:7)	MOVQ	56(SP), DX // 新切片的容量大小
0x00bc 00188 (main.go:7)	ADDQ	$3, CX // 旧切片长度加3，就是新切片的长度
0x00c0 00192 (main.go:7)	JMP	194
0x00c2 00194 (main.go:7)	MOVQ	$5, 16(AX) // 此时才真正append操作，将5写入到底层数组中
0x00ca 00202 (main.go:7)	MOVQ	$6, 24(AX) // 同上
0x00d2 00210 (main.go:7)	MOVQ	$7, 32(AX) // 同上
0x00da 00218 (main.go:7)	PCDATA	$0, $0
0x00da 00218 (main.go:7)	PCDATA	$1, $1
0x00da 00218 (main.go:7)	MOVQ	AX, "".a+128(SP) // 更新切片变量a的底层数组地址
0x00e2 00226 (main.go:7)	MOVQ	CX, "".a+136(SP) // 更新切片变量a的len
0x00ea 00234 (main.go:7)	MOVQ	DX, "".a+144(SP) // 更新切片变量a的cap
```

`runtime.growslice`是go实现的：

```go
// et 是slice元素内存
// old是旧的slice
// cap是新slice最低要求cap大小。是旧的slice的len加上append函数中追加的元素的大小
// 比如s := []int{1,2, 3}；append(s, 4,5)，此时growslice中的cap参数值为5
func growslice(et *_type, old slice, cap int) slice {
	if cap < old.cap {
		panic(errorString("growslice: cap out of range"))
	}

	if et.size == 0 {
		return slice{unsafe.Pointer(&zerobase), old.len, cap}
	}

	newcap := old.cap
	doublecap := newcap + newcap
	if cap > doublecap { // 最小cap要求大于旧slice的cap两倍大小
		newcap = cap
	} else {
		if old.len < 1024 { // 当旧slice的len小于1024, 双倍扩容
			newcap = doublecap
		} else { // 否则每次扩容25%
			for 0 < newcap && newcap < cap {
				newcap += newcap / 4
			}
			if newcap <= 0 {
				newcap = cap
			}
		}
	}

	var overflow bool
	var lenmem, newlenmem, capmem uintptr
	switch {
	case et.size == 1:
		lenmem = uintptr(old.len)
		newlenmem = uintptr(cap)
		capmem = roundupsize(uintptr(newcap))
		overflow = uintptr(newcap) > maxAlloc
		newcap = int(capmem) // 调整newcap大小
	case et.size == sys.PtrSize:
		lenmem = uintptr(old.len) * sys.PtrSize
		newlenmem = uintptr(cap) * sys.PtrSize
		capmem = roundupsize(uintptr(newcap) * sys.PtrSize)
		overflow = uintptr(newcap) > maxAlloc/sys.PtrSize
		newcap = int(capmem / sys.PtrSize)
	case isPowerOfTwo(et.size):
		var shift uintptr
		if sys.PtrSize == 8 {
			// Mask shift for better code generation.
			shift = uintptr(sys.Ctz64(uint64(et.size))) & 63
		} else {
			shift = uintptr(sys.Ctz32(uint32(et.size))) & 31
		}
		lenmem = uintptr(old.len) << shift
		newlenmem = uintptr(cap) << shift
		capmem = roundupsize(uintptr(newcap) << shift)
		overflow = uintptr(newcap) > (maxAlloc >> shift)
		newcap = int(capmem >> shift)
	default:
		lenmem = uintptr(old.len) * et.size
		newlenmem = uintptr(cap) * et.size
		capmem, overflow = math.MulUintptr(et.size, uintptr(newcap))
		capmem = roundupsize(capmem)
		newcap = int(capmem / et.size)
	}

	if overflow || capmem > maxAlloc {
		panic(errorString("growslice: cap out of range"))
	}

	var p unsafe.Pointer
	if et.ptrdata == 0 {
		p = mallocgc(capmem, nil, false)
		memclrNoHeapPointers(add(p, newlenmem), capmem-newlenmem)
	} else {
		p = mallocgc(capmem, et, true)
		if lenmem > 0 && writeBarrier.enabled {
			bulkBarrierPreWriteSrcOnly(uintptr(p), uintptr(old.array), lenmem)
		}
	}
	// 涉及到slice扩容都会有内存移动操作，所以slice一个优化点就是提前设置好cap大小，防止扩容
	memmove(p, old.array, lenmem)

	return slice{p, old.len, newcap}
}
```

结合汇编和Go源码，画出append过程栈帧变化图：

完整带注释的汇编代码：

```as
"".main STEXT size=497 args=0x0 locals=0xd8
	0x0000 00000 (main.go:5)	TEXT	"".main(SB), ABIInternal, $216-0 // 216是main函数的栈帧大小， 0是参数以及返回值的小
	0x0000 00000 (main.go:5)	MOVQ	(TLS), CX // 本地线程存储，不用管
	0x0009 00009 (main.go:5)	LEAQ	-88(SP), AX
	0x000e 00014 (main.go:5)	CMPQ	AX, 16(CX)
	0x0012 00018 (main.go:5)	PCDATA	$0, $-2
	0x0012 00018 (main.go:5)	JLS	487 // 栈自动伸缩处理的逻辑，不用管
	0x0018 00024 (main.go:5)	PCDATA	$0, $-1
	0x0018 00024 (main.go:5)	SUBQ	$216, SP // 分配216字节栈空间
	0x001f 00031 (main.go:5)	MOVQ	BP, 208(SP) //  把调用函数基址入栈
	0x0027 00039 (main.go:5)	LEAQ	208(SP), BP // 把被调用函数即main函数的基址存入BP寄存器
	0x002f 00047 (main.go:5)	PCDATA	$0, $-2 // PCDATA，FUNCDATA 保存了垃圾相关信息，不影响主逻辑，忽略
	0x002f 00047 (main.go:5)	PCDATA	$1, $-2
	0x002f 00047 (main.go:5)	FUNCDATA	$0, gclocals·0ce64bbc7cfa5ef04d41c861de81a3d7(SB)
	0x002f 00047 (main.go:5)	FUNCDATA	$1, gclocals·20f1609b7a8896a9b6839558f8debf33(SB)
	0x002f 00047 (main.go:5)	FUNCDATA	$2, gclocals·f6aec3988379d2bd21c69c093370a150(SB)
	0x002f 00047 (main.go:5)	FUNCDATA	$3, "".main.stkobj(SB)
	0x002f 00047 (main.go:6)	PCDATA	$0, $0
	0x002f 00047 (main.go:6)	PCDATA	$1, $0
	0x002f 00047 (main.go:6)	XORPS	X0, X0
	0x0032 00050 (main.go:6)	MOVUPS	X0, ""..autotmp_6+80(SP)
	0x0037 00055 (main.go:6)	PCDATA	$0, $1
	0x0037 00055 (main.go:6)	LEAQ	""..autotmp_6+80(SP), AX // 把80(SP) 地址加载到AX寄存器， AX寄存器存的是切片a底层数组的地址
	0x003c 00060 (main.go:6)	MOVQ	AX, ""..autotmp_4+112(SP) // 把AX中值加载到120(SP)
	0x0041 00065 (main.go:6)	TESTB	AL, (AX) // 无关指令，不用关心
	0x0043 00067 (main.go:6)	MOVQ	$1, ""..autotmp_6+80(SP) // 切片a第一个元素1加载到数组中
	0x004c 00076 (main.go:6)	TESTB	AL, (AX)
	0x004e 00078 (main.go:6)	MOVQ	$3, ""..autotmp_6+88(SP) // 切片a第二个元素3加载到数组中
	0x0057 00087 (main.go:6)	TESTB	AL, (AX)
	0x0059 00089 (main.go:6)	JMP	91
	0x005b 00091 (main.go:6)	MOVQ	AX, "".a+128(SP) // 切片a的底层数组地址: array
	0x0063 00099 (main.go:6)	MOVQ	$2, "".a+136(SP) // 切片a的长度: len
	0x006f 00111 (main.go:6)	MOVQ	$2, "".a+144(SP) // 切片a的容器: cap
	0x007b 00123 (main.go:7)	JMP	125
	0x007d 00125 (main.go:7)	PCDATA	$0, $2
	0x007d 00125 (main.go:7)	LEAQ	type.int(SB), CX // 把int类型定义加载到CX中
	0x0084 00132 (main.go:7)	PCDATA	$0, $1
	0x0084 00132 (main.go:7)	MOVQ	CX, (SP) // CX加载到(SP)，即(SP) = type.int(SB)，(SP)存储的是切片a的元素类型
	0x0088 00136 (main.go:7)	PCDATA	$0, $0
	0x0088 00136 (main.go:7)	MOVQ	AX, 8(SP) // 切片a的底层数组地址
	0x008d 00141 (main.go:7)	MOVQ	$2, 16(SP) // 切片a的len
	0x0096 00150 (main.go:7)	MOVQ	$2, 24(SP) // 切片a的cap
	0x009f 00159 (main.go:7)	MOVQ	$5, 32(SP) // 新容量大小
	0x00a8 00168 (main.go:7)	CALL	runtime.growslice(SB)
	0x00ad 00173 (main.go:7)	PCDATA	$0, $1
	0x00ad 00173 (main.go:7)	MOVQ	40(SP), AX // 40(SP)存储的是growslice函数返回的底层数组地址， 即append返回的新切片的底层数组地址
	0x00b2 00178 (main.go:7)	MOVQ	48(SP), CX // 旧切片的长度，注意growslice返回的不是新切片的长度，而是旧的切片长度
	0x00b7 00183 (main.go:7)	MOVQ	56(SP), DX // 新切片的容量大小
	0x00bc 00188 (main.go:7)	ADDQ	$3, CX // 旧切片长度加3，就是新切片的长度
	0x00c0 00192 (main.go:7)	JMP	194
	0x00c2 00194 (main.go:7)	MOVQ	$5, 16(AX) // 此时才真正append操作，将5写入到底层数组中
	0x00ca 00202 (main.go:7)	MOVQ	$6, 24(AX) // 同上
	0x00d2 00210 (main.go:7)	MOVQ	$7, 32(AX) // 同上
	0x00da 00218 (main.go:7)	PCDATA	$0, $0
	0x00da 00218 (main.go:7)	PCDATA	$1, $1
	0x00da 00218 (main.go:7)	MOVQ	AX, "".a+128(SP) // 更新切片变量a的底层数组地址
	0x00e2 00226 (main.go:7)	MOVQ	CX, "".a+136(SP) // 更新切片变量a的len
	0x00ea 00234 (main.go:7)	MOVQ	DX, "".a+144(SP) // 更新切片变量a的cap
```