package main

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

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
