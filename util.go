package main

import (
	"io"
	"net"
)

func chkfatal(err error) {
	if err != nil {
		panic(err)
	}
}

func copyClose(a, b net.Conn) {
	done := make(chan struct{})
	oneWay := func(dst, src net.Conn) {
		io.Copy(dst, src)
		dst.Close()
	}
	go oneWay(a, b)
	go oneWay(b, a)
	<-done
	<-done
}
