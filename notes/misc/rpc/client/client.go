package main

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
