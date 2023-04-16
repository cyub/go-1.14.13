package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"go-1.14.13/notes/misc/rpc/proto"
)

type Arith int

func (t *Arith) Multiply(args *proto.Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func (t *Arith) Divide(args *proto.Args, quo *proto.Quotient) error {
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
