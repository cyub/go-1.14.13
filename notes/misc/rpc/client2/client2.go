package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc/jsonrpc"

	"go-1.14.13/notes/misc/rpc/proto"
)

func main() {
	conn, err := net.Dial("tcp", ":8008")
	if err != nil {
		log.Fatal("dial error:", err)

	}

	defer conn.Close()
	client := jsonrpc.NewClient(conn)
	// Synchronous call 同步调用
	args := proto.Args{A: 7, B: 8} // 第一个参数可以是指针类型，也可以不是
	var reply int
	err = client.Call("Arith.Multiply", args, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	fmt.Printf("Arith: %d*%d=%d\n", args.A, args.B, reply)

	// Asynchronous call 异步调用
	for i := 0; i < 5; i++ {
		quotient := new(proto.Quotient)
		args := &proto.Args{A: i, B: i * (i + 1)}
		divCall := client.Go("Arith.Divide", args, quotient, nil)
		call := <-divCall.Done
		if call.Error != nil {
			log.Printf("Arith.Divide quotient: %+v, args: %+v error: %v\n", quotient, args, call.Error)
			continue
		}

		fmt.Printf("Divide Quo: %d/%d=%d\t Divide Rem: %d%%%d=%d\n", args.A, args.B, quotient.Quo, args.A, args.B, quotient.Rem)
	}
}
