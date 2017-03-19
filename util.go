package main

import (
	"io"
	"net"
)

// panic if err is not nil.
func chkfatal(err error) {
	if err != nil {
		panic(err)
	}
}

// Copy data both ways between a and b until one
// of them is closed, then make sure both are closed.
func copyClose(a, b net.Conn) {
	done := make(chan struct{})
	oneWay := func(dst, src net.Conn) {
		io.Copy(dst, src)
		dst.Close()
		done <- struct{}{}
	}
	go oneWay(a, b)
	go oneWay(b, a)
	<-done
	<-done
}
