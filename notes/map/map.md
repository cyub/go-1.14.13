# 映射

映射也被称为哈希表(hash table)、字典。它是一种由key-value组成的抽象数据结构。大多数情况下，它都能在O(1)的时间复杂度下实现增删改查功能。若在极端情况下出现所有key都发生哈希碰撞时则退回成链表形式，此时复杂度为O(N)。映射底层一般都是由数组组成，该数组每个元素称为桶，它使用hash函数将key分配到不同桶中，若出现碰撞冲突时候，则采用链地址法（也称为拉链法）或者开放寻址法解决冲突。下图就是一个由姓名-号码构成的哈希表的结构图：

![](https://static.cyub.vip/images/202106/hash_table.svg)

Go语言中映射中key若出现冲突碰撞时候，则采用链地址法解决，Go语言中映射具有以下特点：

- 引用类型变量
- 读写并发不安全
- 遍历结果是随机的


## 数据结构

Go语言中映射的数据结构是`runtime.hmap`([runtime/map.go](https://github.com/golang/go/blob/go1.14.13/src/runtime/map.go#L115-L129)):

```go
// A header for a Go map.
type hmap struct {
	count     int //  元素个数，用于len函数返回map元素数量
	flags     uint8 // 标志位，标志当前map正在写或者更新等状态
	B         uint8  // buckets个数的对数，即桶数量 = 2 ^ B
	noverflow uint16 // overflow桶数量的近似值，overflow桶即溢出桶，即链表法中存在链表上的桶的个数
	hash0     uint32 // 随机数种子，用于计算key的哈希值

	buckets    unsafe.Pointer // 指向buckets数组，如果元素个数为0时，该值为nil
	oldbuckets unsafe.Pointer // 扩容时指向旧的buckets
	nevacuate  uintptr        // 用于指示迁移进度，小于此值的桶已经迁移完成
	
	extra *mapextra // 额外记录overflow桶信息
}
```

映射中每一个桶的结构是`runtime.bmap`([runtime/map.go](https://github.com/golang/go/blob/go1.14.13/src/runtime/map.go#L149-L159))：

```go
// A bucket for a Go map.
type bmap struct {
	tophash [bucketCnt]uint8
}
```

上面bmap结构是静态结构，在编译过程中`runtime.bmap`会拓展成以下结构体：

```go
type bmap struct{
	tophash [8]uint8
	keys [8]keytype
	values [8]elemtype
	overflow uintptr
}
```

每个桶bmap中可以装载8个key-value键值对。当一个key确定存储在哪个桶之后，还需要确定具体存储在桶的哪个位置（这个位置也称为桶单元，装载8个key-value键值对，那么一个桶共8个桶单元），bmap中tophash就是用于完成这个工作的，实现快速定位key的位置。在实现过程中会使用key的hash值的高八位作为tophash值，存放在bmap的tophash字段中。tophash计算公式如下：
```go
func tophash(hash uintptr) uint8 {
	top := uint8(hash >> (sys.PtrSize*8 - 8))
	if top < minTopHash {
		top += minTopHash
	}
	return top
}
```

上面函数中hash是64位的，sys.PtrSize值是8，所以`top := uint8(hash >> (sys.PtrSize*8 - 8))`等效`top = uint8(hash >> 56)`，最后top取出来的值就是hash的最高8位值。bmap的tophash字段不光存储key哈希值的高八位，还会存储一些状态值，用来表明当前桶单元状态，这些状态值都是小于minTopHash的。为了避免key哈希值的高八位值出现这些状态值相等产生混淆情况，所以当key哈希值高八位若小于时候，自动将其值加上minTopHash作为该key的tophash。桶单元的状态值如下：

```go
emptyRest      = 0 // 表明此桶单元为空，且更高索引的单元也是空
emptyOne       = 1 // 表明此桶单元为空
evacuatedX     = 2 // 用于表示扩容迁移到新桶前半段区间
evacuatedY     = 3 // 用于表示扩容迁移到新桶后半段区间
evacuatedEmpty = 4 // 用于表示此单元已迁移
minTopHash     = 5 // key的tophash值与桶状态值分割线值，小于此值的一定代表着桶单元的状态，大于此值的一定是key对应的tophash值
```

bmap中可以装载8个key-value，这8个key-value并不是按照key1/value1/key2/value2/key3/value3...这样形式存储，而采用key1/key2../key8/value1/../value8形式存储，因为第二种形式可以减少padding，源码中以map[int64]int8举例说明。

hmap中extra字段是`runtime.mapextra`类型，用来记录额外信息：

```go
// mapextra holds fields that are not present on all maps.
type mapextra struct {
	overflow    *[]*bmap
	oldoverflow *[]*bmap

	//指向下一个可用的overflow 桶
	nextOverflow *bmap
}
```

最后我们画出映射的数据结构图：

![](https://static.cyub.vip/images/202106/map_struct.png)

## map的创建

当使用make函数创建映射时候，若不指定map元素数量时候，底层将使用是`make_small`函数创建hmap结构，此时只产生哈希种子，不初始化桶：

```go
func makemap_small() *hmap {
	h := new(hmap)
	h.hash0 = fastrand()
	return h
}
```

若指定map元素数量时候，底层会使用`makemap`函数创建hmap结构：

```go
func makemap(t *maptype, hint int, h *hmap) *hmap {
	mem, overflow := math.MulUintptr(uintptr(hint), t.bucket.size)
	if overflow || mem > maxAlloc { // 检查所有桶占用的内存是否大于内存限制
		hint = 0
	}

	// h不nil，说明map结构已经创建在栈上了，这个操作由编译器处理的
	if h == nil { // h为nil，则需要创建一个hmap类型
		h = new(hmap)
	}
	h.hash0 = fastrand() // 设置map的随机数种子

	B := uint8(0)
	for overLoadFactor(hint, B) { // 设置合适B的值
		B++
	}
	h.B = B

    // 如果B == 0，那么map的buckets，将会惰性分配（allocated lazily），使用时候再分配
    // 如果B != 0时，初始化桶
	if h.B != 0 {
		var nextOverflow *bmap
		h.buckets, nextOverflow = makeBucketArray(t, h.B, nil)
		if nextOverflow != nil {
			h.extra = new(mapextra)
			h.extra.nextOverflow = nextOverflow // extra.nextOverflow指向下一个可用溢出桶位置
		}
	}

	return h
}
```

makemap函数的第一个参数是maptype类指针，它描述了创建的map中key和value元素的类型信息以及其他map信息，第二个参数hint，对应是`make([Type]Type, len)`中len参数，第三个参数h，如果不为nil，说明当前map的结构已经有编译器在栈上创建了，makemap只需要完成设置随机数种子等操作。

`overLoadFactor`函数用来判断当前映射大小是否超过加载因子。`makemap`使用`overLoadFactor`函数来调整B值。加载因子描述了哈希表中元素填满程度，加载因子越大，表明哈希表中元素越多，空间利用率高，但是这也意味着冲突的机会就会加大。当哈希表中所有桶已写满情况下，加载因子就是1，此时再写入新key一定会产生冲突碰撞。为了提高哈希表写入效率就必须在加载因子超过一定值时，进行rehash操作，将桶容量进行扩容，来尽量避免出现冲突情况。Java中hashmap的默认加载因子是0.75，Go语言中映射的加载因子是6.5。为什么Go映射的加载因子超过了1，这是因为Go映射中每个桶可以存8个key-value，而一般哈希表都是存放一个key-value，其满载因子是1，而Go则是8。

```go
func overLoadFactor(count int, B uint8) bool {
    // count > bucketCnt，bucketCnt值是8，每一个桶可以存放8个key-value，如果map中元素个数count小于8那么一定不会超过加载因子

    // loadFactorNum和loadFactorDen的值分别是13和2，bucketShift(B)等效于1<<B
    // 所以 uintptr(count) > loadFactorNum*(bucketShift(B)/loadFactorDen) 等于  uintptr(count) > 6.5 * 2^ B
	return count > bucketCnt && uintptr(count) > loadFactorNum*(bucketShift(B)/loadFactorDen)
}

// bucketShift returns 1<<b, optimized for code generation.
func bucketShift(b uint8) uintptr {
	return uintptr(1) << (b & (sys.PtrSize*8 - 1))
}
```

`makeBucketArray`函数是用来创建array，来用作为map的buckets。对于创建时指定元素大小超过(2^4) * 8时候，除了创建map的buckets，也会提前分配好一些桶作为溢出桶。buckets和溢出桶，在内存上是连续的。为啥提前分配好溢出桶，而不是在溢出时候，再分配，这是因为现在分配，是直接申请一大片内存，效率更高。hamp.extra.nextOverflow指向该溢出桶，溢出桶的除了最后一个桶的overflow指向map的buckets，其他桶的overflow指向nil，这是用来判断溢出桶最后边界，后面代码有涉及此逻辑。

```go
func makeBucketArray(t *maptype, b uint8, dirtyalloc unsafe.Pointer) (buckets unsafe.Pointer, nextOverflow *bmap) {
	base := bucketShift(b) // 等效于 base := 1 << b
	nbuckets := base

	if b >= 4 { // 对于小b，不太可能出现溢出桶，所以B超过4时候，才考虑提前分配写溢出桶
		nbuckets += bucketShift(b - 4)
		sz := t.bucket.size * nbuckets
		up := roundupsize(sz)
		if up != sz {
			nbuckets = up / t.bucket.size
		}
	}

    // 若dirtyalloc不为nil时，
	// dirtyalloc指向的之前已经使用完的map的buckets，之前已使用完的map和当前map具有相同类型的t和b，这样它buckets可以拿来复用
    // 此时只需对dirtyalloc进行清除操作就可以作为当前map的buckets

	if dirtyalloc == nil {
		buckets = newarray(t.bucket, int(nbuckets))
	} else {
		buckets = dirtyalloc
		size := t.bucket.size * nbuckets
		if t.bucket.ptrdata != 0 { // map中key或value是指针类型
			memclrHasPointers(buckets, size)
		} else {
			memclrNoHeapPointers(buckets, size)
		}
	}

	if base != nbuckets { // 创建一些溢出桶结构体
		nextOverflow = (*bmap)(add(buckets, base*uintptr(t.bucketsize)))
		// 溢出桶的最后一个bmap的overflow指向buckets
		last := (*bmap)(add(buckets, (nbuckets-1)*uintptr(t.bucketsize)))
		last.setoverflow(t, (*bmap)(buckets))
	}
	return buckets, nextOverflow
}
```

我们画出桶分配示意图：


从上面可以看到使用make创建map时候，返回都是hmap类型指针，这也就说明**Go语言中映射时引用类型的**。

## 访问映射操作

访问映射涉及到key定位的问题，首先需要确定从哪个桶找，确定桶之后，还需要确定key-value具体存放在哪个单元里面（比较每个桶里面有8个坑位）。key定位详细流程如下：

1. 首先需根据hash函数计算出key的hash值
2. 该key的hash值的低`hmap.B`位的值是该key所在的桶
3. 该key的hash值的高8位，用来快速定位其在桶具体位置。一个桶中存放8个key，遍历所有key，找到等于该key的位置，此位置对应的就是值所在位置
4. 根据步骤3取到的值，计算该值的hash，再次比较，若相等则定位成功。否则重复步骤3去`bmap.overflow`中继续查找。
5. 若`bmap.overflow`链表都找个遍都没有找到，则返回nil。


当m为2的x幂时候，n对m取余数存在以下等式：

```
n % m = n & (m -1)
```

举个例子比如：n为15，m为8，n%m等7, n&(m-1)也等于7，取余应尽量使用第二种方式，因为效率更高。

对于映射中key定位计算就是：

```
key对应value所在桶位置 = hash(key)%(hmap.B << 1) = hash(key) & (hmap.B <<1 - 1)
```
那么为什么上面key定位流程步骤2中说的却是根据该key的hash值的低`hmap.B`位的值是该key所在的桶。两者是没有区别的，只是一种意思不同说法。


### 直接访问与逗号ok模式访问

访问映射操作方式有两种：

第一种直接访问，若key不存在，则返回value类型的零值，其底层实现`mapaccess1`函数：

```go
v := a["x"]
```
第二种是逗号ok模式，如果key不存在，除了返回value类型的零值，ok变量也会设置为false，其底层实现`mapaccess2`：

```go
v, ok := a["x"]
```

为了优化性能，Go编译器会根据key类型采用不同底层函数，比如对于key类型是int的，底层实现是mapaccess1_fast64。具体文件可以查看runtime/map_fastxxx.go。这里面我们这分析通用的mapaccess1函数。

```go
func mapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	if h == nil || h.count == 0 { // map中元素个数为0，则直接返回零值
		if t.hashMightPanic() {
			t.hasher(key, 0) // see issue 23734
		}
		return unsafe.Pointer(&zeroVal[0])
	}
	if h.flags&hashWriting != 0 { // 有其他Goroutine正在写map，则直接panic
		throw("concurrent map read and map write")
	}
	hash := t.hasher(key, uintptr(h.hash0)) // 计算出key的hash值
	m := bucketMask(h.B) // m = 2^B - 1
	b := (*bmap)(add(h.buckets, (hash&m)*uintptr(t.bucketsize))) // 根据上面介绍的取余操作转换成位与操作来获取key所在的桶
	if c := h.oldbuckets; c != nil { // 如果oldbuckets不为0，说明该map正在处于扩容过程中
		if !h.sameSizeGrow() { // 如果不是等容量扩容，此时buckets大小是oldbuckets的两倍，那么m需减半，然后用来定位key在旧桶中位置
			m >>= 1
		}
		oldb := (*bmap)(add(c, (hash&m)*uintptr(t.bucketsize))) // 获取key在旧桶的桶
		if !evacuated(oldb) { // 如果旧桶数据没有迁移新桶里面，那就在旧桶里面找
			b = oldb
		}
	}
	top := tophash(hash) // 计算出key的tophash
bucketloop:
	for ; b != nil; b = b.overflow(t) { // for循环实现功能是先从当前桶找，若未找到则当前桶的溢出桶b.overfolw(t)查找，直到溢出桶为nil
		for i := uintptr(0); i < bucketCnt; i++ { // 每个桶有8个单元，循环这8个单元，一个个找
			if b.tophash[i] != top { // 如果当前单元的tophash与key的tophash不一致，
				if b.tophash[i] == emptyRest { // 若单元tophash值是emptyRest，则直接跳出整个大循环，emptyRest表明当前单元和更高单元存储都为空，所以无需在继续查找下去了
					break bucketloop
				}
				continue // 继续查找桶其他的单元
			}

            // 此时已找到tophash等于key的tophash的桶单元，此时i记录这桶单元编号
			k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize)) // dataOffset是bmap.keys相对于bmap的偏移，k记录key存在bmap的位置
			if t.indirectkey() { // 若key是指针类型
				k = *((*unsafe.Pointer)(k))
			}
			if t.key.equal(key, k) {// 如果key和存放bmap里面的key相等则获取对应value值返回
                // value在bmap中的位置 = bmap.keys相对于bmap的偏移 + 8个key占用的空间(8 * keysize) + 该value在bmap.values中偏移(i * t.elemsize)
				e := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
				if t.indirectelem() {
					e = *((*unsafe.Pointer)(e))
				}
				return e
			}
		}
	}
	return unsafe.Pointer(&zeroVal[0])
}
```

## 赋值操作

在map中增加和更新key-value时候，都会调用`runtime.mapassign`方法，同访问方法一样，Go编译器针对不同类型的key，会采用优化版本函数：

key 类型 | 方法
---- | ----
uint64 | func mapassign_fast64(t *maptype, h *hmap, key uint64) unsafe.Pointer
unsafe.Pointer | func mapassign_fast64ptr(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer
string | func mapassign_faststr(t *maptype, h *hmap, s string) unsafe.Pointer
uint32 | func mapassign_fast32(t *maptype, h *hmap, key uint32) unsafe.Pointer
unsafe.Pointer | func mapassign_fast32ptr(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer

这里面我们只分析通用的方法：

```go
func mapassign(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
	if h == nil {
		panic(plainError("assignment to entry in nil map"))
	}
	if raceenabled {
		callerpc := getcallerpc()
		pc := funcPC(mapassign)
		racewritepc(unsafe.Pointer(h), callerpc, pc)
		raceReadObjectPC(t.key, key, callerpc, pc)
	}
	if msanenabled {
		msanread(key, t.key.size)
	}
	if h.flags&hashWriting != 0 { // 有其他Goroutine的写操作正在进行中，则直接panic
		throw("concurrent map writes")
	}
	hash := t.hasher(key, uintptr(h.hash0))

	// Set hashWriting after calling t.hasher, since t.hasher may panic,
	// in which case we have not actually done a write.
	h.flags ^= hashWriting // 将写标志位置为1

	if h.buckets == nil {
		h.buckets = newobject(t.bucket) // newarray(t.bucket, 1)
	}

again:
	bucket := hash & bucketMask(h.B)
	if h.growing() { // 若果还在扩容中，则调用growWork，每次只搬迁2个旧bucket到新bucket中，且保证当前bucket对应的老bucket一定会搬到新的bucket中
		growWork(t, h, bucket)
	}
	b := (*bmap)(unsafe.Pointer(uintptr(h.buckets) + bucket*uintptr(t.bucketsize)))
	top := tophash(hash)

	var inserti *uint8 //tophash地址
	var insertk unsafe.Pointer // key地址
	var elem unsafe.Pointer // 旧value
bucketloop:
	for {
		for i := uintptr(0); i < bucketCnt; i++ {
			if b.tophash[i] != top {
				if isEmpty(b.tophash[i]) && inserti == nil { // 如果当前tophash值是0，说明此处没有使用，则使用此槽位
					inserti = &b.tophash[i]
					insertk = add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
					elem = add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
				}
				if b.tophash[i] == emptyRest {
					break bucketloop
				}
				continue
			}
			k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
			if t.indirectkey() {
				k = *((*unsafe.Pointer)(k))
			}
			if !t.key.equal(key, k) {
				continue
			}
			// already have a mapping for key. Update it.
			if t.needkeyupdate() {
				typedmemmove(t.key, k, key)
			}
			elem = add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
			goto done
		}
		ovf := b.overflow(t)
		if ovf == nil {
			break
		}
		b = ovf
	}

	// Did not find mapping for key. Allocate new cell & add entry.

	// If we hit the max load factor or we have too many overflow buckets,
	// and we're not already in the middle of growing, start growing.
	if !h.growing() && (overLoadFactor(h.count+1, h.B) || tooManyOverflowBuckets(h.noverflow, h.B)) {
		hashGrow(t, h)
		goto again // Growing the table invalidates everything, so try again
	}

	if inserti == nil {
		// all current buckets are full, allocate a new one.
		newb := h.newoverflow(t, b)
		inserti = &newb.tophash[0]
		insertk = add(unsafe.Pointer(newb), dataOffset)
		elem = add(insertk, bucketCnt*uintptr(t.keysize))
	}

	// store new key/elem at insert position
	if t.indirectkey() {
		kmem := newobject(t.key)
		*(*unsafe.Pointer)(insertk) = kmem
		insertk = kmem
	}
	if t.indirectelem() {
		vmem := newobject(t.elem)
		*(*unsafe.Pointer)(elem) = vmem
	}
	typedmemmove(t.key, insertk, key)
	*inserti = top
	h.count++

done:
	if h.flags&hashWriting == 0 {
		throw("concurrent map writes")
	}
	h.flags &^= hashWriting
	if t.indirectelem() {
		elem = *((*unsafe.Pointer)(elem))
	}
	return elem
}
```

## 扩容方式
