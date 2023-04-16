# RPC

RPC是Remote Procedure Call的简称，通过RPC客户端像调用本地函数一样，调用远程服务端的方法。RPC框架底层完成了网络传输，编码解码、路由等工作，提供给使用方简单一致的使用体验。在网络传输层面，RPC跨越了OSI网络模型中的传输层和应用层，使用者不用关心这些。

## Go 内置RPC包

Go 标准包net/rpc提供了对RPC的支持，目前该包处于维护状态，不再接受新特性。该包支持的特性如下：

特性 | Go rpc | Gorilla rpc 
--- | --- | ---
TCP协议 | ✅ | ❎
HTTP协议 | ✅ | ✅
同步调用 | ✅ | ✅
异步调用 | ✅ | ✅
encoding/gob编码 | ✅ | ✅
json-rpc | ✅ | ✅


Go内置RPC默认使用encode/gob进行传输内容的编码和解码，此外还支持json编码和解码，它大致实现json-rpc v2.0功能。

## net/rpc用例

### 基于http协议gob编码实现的rpc

服务端（默认是encode/gob编码）：

```go
import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

type Arith int

func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}

func main() {
	arith := new(Arith)
	rpc.Register(arith)
	rpc.HandleHTTP() // 使用HTTP协议
	l, e := net.Listen("tcp", ":8008")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
	select {}
}
```

客户端：

```go

import (
	"fmt"
	"log"
	"net/rpc"
)

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

func main() {
	serverAddress := "localhost"
	client, err := rpc.DialHTTP("tcp", serverAddress+":8008") // 服务端是HTTP服务，一定要使用DialHTTP
	if err != nil {
		log.Fatal("dialing:", err)
	}

	// Synchronous call 同步调用
	args := Args{7, 8} // 第一个参数可以是指针类型，也可以不是
	var reply int
	err = client.Call("Arith.Multiply", args, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	fmt.Printf("Arith: %d*%d=%d\n", args.A, args.B, reply)

	// Asynchronous call 异步调用
	for i := 0; i < 5; i++ {
		quotient := new(Quotient)
		args := &Args{i, i * (i + 1)}
		divCall := client.Go("Arith.Divide", args, quotient, nil)
		call := <-divCall.Done
		if call.Error != nil {
			log.Printf("Arith.Divide quotient: %+v, args: %+v error: %v\n", quotient, args, call.Error)
			continue
		}

		fmt.Printf("Divide Quo: %d/%d=%d\t Divide Rem: %d%%%d=%d\n", args.A, args.B, quotient.Quo, args.A, args.B, quotient.Rem)
	}
}
```

### 基于tcp协议的json-rpc

服务端：

```go
type Arith int

func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}

func main() {
	ln, err := net.Listen("tcp", ":8008")
	if err != nil {
		log.Fatal("listen error:", err)
	}

	rpc.Register(new(Arith))

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("listen Accept error: ", err)
		}
		log.Printf("%s accepted", conn.RemoteAddr())
		go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
```

客户端：

```go
import (
	"fmt"
	"log"
	"net/rpc"
    "net/rpc/jsonrpc"
)

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

func main() {
	conn, err := net.Dial("tcp", ":8008")
	if err != nil {
		log.Fatal("dial error:", err)

	}

	defer conn.Close()
	client := jsonrpc.NewClient(conn)
	// Synchronous call 同步调用
	args := Args{A: 7, B: 8} // 第一个参数可以是指针类型，也可以不是
	var reply int
	err = client.Call("Arith.Multiply", args, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	fmt.Printf("Arith: %d*%d=%d\n", args.A, args.B, reply)

	// Asynchronous call 异步调用
	for i := 0; i < 5; i++ {
		quotient := new(Quotient)
		args := &Args{A: i, B: i * (i + 1)}
		divCall := client.Go("Arith.Divide", args, quotient, nil)
		call := <-divCall.Done
		if call.Error != nil {
			log.Printf("Arith.Divide quotient: %+v, args: %+v error: %v\n", quotient, args, call.Error)
			continue
		}

		fmt.Printf("Divide Quo: %d/%d=%d\t Divide Rem: %d%%%d=%d\n", args.A, args.B, quotient.Quo, args.A, args.B, quotient.Rem)
	}
}
```

## 资料

- [关于HTTP CONNECT方法](https://www.zhihu.com/tardis/bd/art/533663637)