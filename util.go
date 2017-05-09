package main

import (
	"context"
	"io"
	"net"
)

// panic if err is not nil.
func chkfatal(err error) {
	if err != nil {
		panic(err)
	}
}

// Copy data both ways between a and b until one of them is closed,
// or the context is cancelled, then make sure both are closed.
func copyClose(ctx context.Context, a, b net.Conn) {
	defer a.Close()
	defer b.Close()
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	oneWay := func(dst, src net.Conn) {
		io.Copy(dst, src)
		cancel()
	}
	go oneWay(a, b)
	go oneWay(b, a)
	<-ctx.Done()
}
