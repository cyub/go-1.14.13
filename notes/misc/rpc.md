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


Go内置RPC默认使用`encoding/gob`进行传输内容的编码和解码，此外还支持json编码和解码，但它只实现了json-rpc v1.0功能。

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

## 代码分析

### 服务端

RPC服务端接收请求数据，然后异步处理响应：

```go
// ServeCodec is like ServeConn but uses the specified codec to
// decode requests and encode responses.
func (server *Server) ServeCodec(codec ServerCodec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec) // 读取请求对象
		if err != nil {
			if debugLog && err != io.EOF {
				log.Println("rpc:", err)
			}
			if !keepReading {
				break
			}
			// send a response if we actually managed to read a header.
			if req != nil {
				server.sendResponse(sending, req, invalidRequest, codec, err.Error())
				server.freeRequest(req)
			}
			continue
		}
		wg.Add(1)
		go service.call(server, sending, wg, mtype, req, argv, replyv, codec) // 异步处理请求数据，并响应
	}
	// We've seen that there are no more requests.
	// Wait for responses to be sent before closing codec.
	wg.Wait() // 当客户端主动关闭连接（半关闭）时候，考虑到处理请求是异步的，所以需要一个同步机制来保证所有请求处理完成。
	// 这里使用waitgroup，当前一个请求过来后Add(1)，当一个请求处理完成后Done()
	codec.Close()
}
```

### 客户端

**发送请求**：

客户端`Go`接口串行发送数据，并通过等待通道获取请求响应状态：

```go
func (client *Client) Go(serviceMethod string, args any, reply any, done chan *Call) *Call {
	call := new(Call)
	call.ServiceMethod = serviceMethod
	call.Args = args
	call.Reply = reply
	if done == nil {
		done = make(chan *Call, 10) // buffered.
	} else {
		// If caller passes done != nil, it must arrange that
		// done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		if cap(done) == 0 {
			log.Panic("rpc: done channel is unbuffered")
		}
	}
	call.Done = done // done通道用于等待响应状态，当请求完成，done是可读取状态状态
	client.send(call)
	return call
}

func (client *Client) send(call *Call) {
	client.reqMutex.Lock() // reqMutex锁保证发送数据的顺序
	defer client.reqMutex.Unlock()

	// Register this call.
	client.mutex.Lock()
	if client.shutdown || client.closing {
		client.mutex.Unlock()
		call.Error = ErrShutdown
		call.done()
		return
	}
	seq := client.seq // seq是请求的唯一ID
	client.seq++
	client.pending[seq] = call // 请求ID与请求对象绑定
	client.mutex.Unlock()

	// Encode and send the request.
	client.request.Seq = seq
	client.request.ServiceMethod = call.ServiceMethod
	err := client.codec.WriteRequest(&client.request, call.Args)
	if err != nil {
		client.mutex.Lock()
		call = client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}
```

**响应处理：**

客户端创建时候，专门启动一个`input`协程处理服务端响应：

```go
func NewClientWithCodec(codec ClientCodec) *Client {
	client := &Client{
		codec:   codec,
		pending: make(map[uint64]*Call),
	}
	go client.input()
	return client
}
```

`input`协程会读取服务端响应数据，获取请求ID后找到其关联的请求对象，然后将响应写入到请求对象Reply属性中，并将请求对象的done通道close掉，来通知等待done通道的客户端请求完成：

```go
func (client *Client) input() {
	var err error
	var response Response
	for err == nil {
		response = Response{}
		err = client.codec.ReadResponseHeader(&response)
		if err != nil {
			break
		}
		seq := response.Seq
		client.mutex.Lock()
		call := client.pending[seq] // 获取请求ID对应的请求对象
		delete(client.pending, seq)
		client.mutex.Unlock()

		switch {
		case call == nil:
			// We've got no pending call. That usually means that
			// WriteRequest partially failed, and call was already
			// removed; response is a server telling us about an
			// error reading request body. We should still attempt
			// to read error body, but there's no one to give it to.
			err = client.codec.ReadResponseBody(nil)
			if err != nil {
				err = errors.New("reading error body: " + err.Error())
			}
		case response.Error != "":
			// We've got an error response. Give this to the request;
			// any subsequent requests will get the ReadResponseBody
			// error if there is one.
			call.Error = ServerError(response.Error)
			err = client.codec.ReadResponseBody(nil)
			if err != nil {
				err = errors.New("reading error body: " + err.Error())
			}
			call.done()
		default:
			err = client.codec.ReadResponseBody(call.Reply) // 读取响应数据到call对象中
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	// Terminate pending calls.
	client.reqMutex.Lock()
	client.mutex.Lock()
	client.shutdown = true
	closing := client.closing
	if err == io.EOF {
		if closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
	client.mutex.Unlock()
	client.reqMutex.Unlock()
	if debugLog && err != io.EOF && !closing {
		log.Println("rpc: client protocol error:", err)
	}
}
```

## 资料

- [关于HTTP CONNECT方法](https://www.zhihu.com/tardis/bd/art/533663637)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)