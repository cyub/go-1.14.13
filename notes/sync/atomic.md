## atomic概述

atomic是Go内置原子操作包。下面是官方说明：

> Package atomic provides low-level atomic memory primitives useful for implementing synchronization algorithms.

atomic包提供了用于实现同步机制的底层原子内存原语。

> These functions require great care to be used correctly. Except for special, low-level applications, synchronization is better done with channels or the facilities of the sync package. Share memory by communicating; don't communicate by sharing memory

使用这些功能需要非常小心。除了特殊的底层应用程序外，最好使用通道或sync包来进行同步。**通过通信来共享内存；不要通过共享内存来通信**。

atomic包提供的操作可以分为三类：
### 对整数类型T的操作

T类型是`int32`、`int64`、`uint32`、`uint64`、`uintptr`其中一种。

```go
func AddT(addr *T, delta T) (new T)
func CompareAndSwapT(addr *T, old, new T) (swapped bool)
func LoadT(addr *T) (val T)
func StoreT(addr *T, val T)
func SwapT(addr *T, new T) (old T)
```

### 对于`unsafe.Pointer`类型的操作

```go
func CompareAndSwapPointer(addr *unsafe.Pointer, old, new unsafe.Pointer) (swapped bool)
func LoadPointer(addr *unsafe.Pointer) (val unsafe.Pointer)
func StorePointer(addr *unsafe.Pointer, val unsafe.Pointer)
func SwapPointer(addr *unsafe.Pointer, new unsafe.Pointer) (old unsafe.Pointer)
```

### `atomic.Value`类型提供Load/Store操作

atomic提供了`atomic.Value`类型，用来原子性加载和存储类型一致的值（consistently typed value）。

```go
func (v *Value) Load() (x interface{}) // 原子性返回刚刚存储的值，若没有值返回nil
func (v *Value) Store(x interface{}) // 原子性存储值x，x可以是nil，但需要每次存的值都必须是同一个具体类型。
```

## 用法

用法示例1：原子性增加值

```go
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func main() {
	var count int32
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			atomic.AddInt32(&count, 1) // 原子性增加值
			wg.Done()
		}()
		go func() {
			fmt.Println(atomic.LoadInt32(&count)) // 原子性加载
		}()
	}
	wg.Wait()
	fmt.Println("count: ", count)
}
```