# 术语

## 机器语言

机器语言是机器指令的集合。计算机的机器指令是一系列二进制数字。计算机将之转换为一系列高低电平脉冲信号来驱动硬件工作的

## 汇编语言

机器指令是由0和1组成的二进制指令，难以编写与记忆。汇编语言是二进制指令的文本形式，与机器指令一一对应，相当于机器指令的助记码。比如，加法的机器指令是`00000011`写成汇编语言就是`ADD`。

将助记码标准化后称为`assembly language`，缩写为`asm`，中文译为汇编语言

汇编语言大致可以分为两类：

1. 基于x86架构处理器的汇编语言

    - Intel 汇编
        - DOS(8086处理器), Windows
        - Windows 派系 -> VC 编译器
    - AT&T汇编
        - Linux, Unix, Mac OS, iOS(模拟器)
        - Unix派系 -> GCC编译器
2. 基于ARM 架构处理器的汇编语言

    - ARM 汇编

### 数据单元大小

汇编中数据单元大小可分为：

- 位 bit
- 半字节 Nibble
- 字节 Byte
- 字 Word 相当于两个字节
- 双字 Double Word 相当于2个字，4个字节
- 四字 Quadword 相当于4个字，8个字节

## 寄存器

寄存器是CPU中存储数据的器件，起到数据缓存作用。缓存按照层级依次分为寄存器，L1 Cache, L2 Cache, L3 Cache，其读写延迟依次增加，实现成本依次降低。

![](https://static.cyub.vip/images/202007/cpu_cache.png)

### 寄存器分类

一个CPU中有多个寄存器。每一个寄存器都有自己的名称。寄存器按照种类分为通用寄存器和控制寄存器。其中通用寄存器有可细分为数据寄存器，指针寄存器，以及变址寄存器。

![](https://static.cyub.vip/images/202007/register.jpg)

1979年因特尔推出8086架构的CPU，开始支持16位。为了兼容之前8008架构的8位CPU，8086架构中AX寄存器高8位称为AH，低8位称为AL，用来对应8008架构的8位的A寄存器。后来随着x86，以及x86-64
架构的CPU推出，开始支持32位以及64位，为了兼容并保留了旧名称，16位处理器的AX寄存器拓展成EAX(E代表拓展Extended的意思)。对于64位处理器的寄存器相应的RAX(R代表寄存器Register的意思)。其他指令也类似。

![](https://static.cyub.vip/images/202102/rax.jpg)

寄存器 | 功能
---|---
**AX**| A代表累加器Accumulator，X是八位寄存器AH和AL的中H和L的占位符，表示AX由AH和AL组成。AX一般用于算术与逻辑运算，以及作为函数返回值
**BX** | B代表Base，BX一般用于保存中间地址(hold indirect addresses)
**CX** | C代表Count，CX一般用于计数，比如使用它来计算循环中的迭代次数或指定字符串中的字符数
**DX** | D代表Data，DX一般用于保存某些算术运算的溢出，并且在访问80x86 I/O总线上的数据时保存I/O地址
**DI** | DI代表Destination Index，DI一般用于指针
**SI** | SI代表Source Index，SI用途同DI一样
**SP** | SP代表Stack Pointer，是栈指针寄存器，存放着执行函数对应栈帧的栈顶地址，且始终指向栈顶
**BP** | BP代表Base Pointer，是栈帧基址指针寄存器，存放这执行函数对应栈帧的栈底地址，一般用于访问栈中的局部变量和参数
**IP** | IP代表Instruction Pointer，是指令寄存器，指向处理器下条等待执行的指令地址(代码段内的偏移量)，每次执行完相应汇编指令IP值就会增加；IP是个特殊寄存器，不能像访问通用寄存器那样访问它。IP可被jmp、call和ret等指令隐含地改变


## CPU对存储器的读写

CPU要对数据进行读写，必须和外部器件进行以下三类信息的交互：

1. 存储单元的地址(地址信息)
2. 器件的选择、读或写命令(控制信息)
3. 读或写的数据(数据信息) 

总线是连接CPU和其他芯片的导线，逻辑上分为地址总线、数据总线、控制总线

![](https://static.cyub.vip/images/202102/cpu_bus.webp)

CPU从内存单元中读写数据的过程：

1. CPU通过地址线将地址信息发出；
2. CPU通过控制线发出内存读命令，选中存储器芯片，并通知它将要从中读或写数据；
3. 存储器将相应的地址单元中的数据通过数据线送入CPU或CPU通过数据线将数据送入相应的内存单元

![](https://static.cyub.vip/images/202102/cpu_struct.gif)

### 地址总线

CPU是通过地址总线指定存储单元，地址总线传送的能力决定了CPU对存储单元的寻址能力。对于32位CPU，其寻址能力为2^32=4G。

地址寄存器存储的是CPU当前要存取的数据或指令的地址，该地址是由地址总线传输到地址寄存器上的。

### 数据总线

CPU通过数据总线来与内存等器件进行数据传送，数据总线的宽度决定了CPU和外界的数据传送速度。

### 控制总线

控制总线是一些不同控制的集合，CPU通过控制总线对外部器件的控制。控制总线的宽度决定了CPU对外部器件的控制能力。

### CPU位数

CPU位数与地址寄存器位数，以及数据总线的宽度是一致的。

地址总线的宽度并不一定与CPU位数一致。目前各种架构的64位CPU通常是42条地址线。


## 编译流程

## 应用程序在虚拟内存中的布局

当应用程序运行起来时候，系统会将该应用加载到内存中，应用会独立的、完全的占用内存，该内存并不是物理内存，而是虚拟内存，对于32位系统，该虚拟内存大小是2^32 = 4G。操作系统会完成虚拟内存到物理内存的映射处理工作，应用程序并不需要关心。进程加载到虚拟内存中，这就牵扯到进程在虚拟内存的布局。

进程在内存布局分为以下几大块
- Stack - 栈
- Heap - 堆
- BSS - 未初始化数据区，对应的汇编是(.section .bss)
- DS - 初始化化数据区, 对应的汇编是(.section .data)
- Text - 文本区，程序代码, 对应的汇编是(.section .text)

内存布局简图：

```
High Addresses ---> .----------------------.
                    |      Environment     |
                    |----------------------|
                    |                      |   Functions and variable are declared
                    |         STACK        |   on the stack.
base pointer ->     | - - - - - - - - - - -|
                    |           |          |
                    |           v          |
                    :                      :
                    .                      .   The stack grows down into unused space
                    .         Empty        .   while the heap grows up. 
                    .                      .
                    .                      .   (other memory maps do occur here, such 
                    .                      .    as dynamic libraries, and different memory
                    :                      :    allocate)
                    |           ^          |
                    |           |          |
 brk point ->       | - - - - - - - - - - -|   Dynamic memory is declared on the heap
                    |          HEAP        |
                    |                      |
                    |----------------------|
                    |          BSS         |   Uninitialized data (BSS)
                    |----------------------|   
                    |          Data        |   Initialized data (DS)
                    |----------------------|
                    |          Text        |   Binary code
Low Addresses ----> '----------------------'
```

详细图：
![](https://static.cyub.vip/images/202102/process_mem_layout.jpeg)


## 寻址方式

寻址方式就是CPU根据指令中给出的地址信息来寻找有效地址的方式，是确定本条指令的数据地址以及下一条要执行的指令地址的方法

寻址方式 | 寻址指令 | 解释
---|---|---
立即寻址 | movl $number, %eax | 将number直接存储到到寄存器或存储位置
直接寻址 | movl 0x123, %eax | 将内存地址0x123存储到到%eax寄存器中
索引寻址/变址寻址方式 | movl string_start(, %ecx, 1), %eax | 将string_start地址与1 * %ecx相加得到新地址，从该新地址加载数据到%eax寄存器中
间接寻址方式 | mov (%eax), %ebx | 从寄存器%eax中存储的地址加载值到%ebx寄存器中
基址寻址方式 | movl 4(%eax), %ebx | 将寄存器%eax中存储的地址加上4字节后得到地址，从该地址加载数据到寄存器%ebx中

寻址方式其实就是传递数据的方式。传递数据的大小由mov指令的后缀决定。

```as
movb $123, %eax // 1 byte
movw $123, %eax // 2 byte
movl $123, %eax // 4 byte
movq $123, %eax // 8 byte
```

## 栈帧
## PC

# Plan9 汇编

## 函数调用栈图

```
                                                                                                                              
                                       caller                                                                                 
                                 +------------------+                                                                         
                                 |                  |                                                                         
       +---------------------->  --------------------                                                                         
       |                         |                  |                                                                         
       |                         | caller parent BP |                                                                         
       |           BP(pseudo SP) --------------------                                                                         
       |                         |                  |                                                                         
       |                         |   Local Var0     |                                                                         
       |                         --------------------                                                                         
       |                         |                  |                                                                         
       |                         |   .......        |                                                                         
       |                         --------------------                                                                         
       |                         |                  |                                                                         
       |                         |   Local VarN     |                                                                         
                                 --------------------                                                                         
 caller stack frame              |                  |                                                                         
                                 |   callee arg2    |                                                                         
       |                         |------------------|                                                                         
       |                         |                  |                                                                         
       |                         |   callee arg1    |                                                                         
       |                         |------------------|                                                                         
       |                         |                  |                                                                         
       |                         |   callee arg0    |                                                                         
       |                         ----------------------------------------------+   FP(virtual register)                       
       |                         |                  |                          |                                              
       |                         |   return addr    |  parent return address   |                                              
       +---------------------->  +------------------+---------------------------    <-------------------------------+         
                                                    |  caller BP               |                                    |         
                                                    |  (caller frame pointer)  |                                    |         
                                     BP(pseudo SP)  ----------------------------                                    |         
                                                    |                          |                                    |         
                                                    |     Local Var0           |                                    |         
                                                    ----------------------------                                    |         
                                                    |                          |                                              
                                                    |     Local Var1           |                                              
                                                    ----------------------------                            callee stack frame
                                                    |                          |                                              
                                                    |       .....              |                                              
                                                    ----------------------------                                    |         
                                                    |                          |                                    |         
                                                    |     Local VarN           |                                    |         
                                  SP(Real Register) ----------------------------                                    |         
                                                    |                          |                                    |         
                                                    |                          |                                    |         
                                                    |                          |                                    |         
                                                    |                          |                                    |         
                                                    |                          |                                    |         
                                                    +--------------------------+    <-------------------------------+         
                                                                                                                              
                                                              callee
```


# 资料

- [plan9 assembly 完全解析](https://github.com/cch123/golang-notes/blob/master/assembly.md)
- [EAX x86 Register Meaning and History](https://keleshev.com/eax-x86-register-meaning-and-history/)