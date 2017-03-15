package main

import (
	"context"
	"log"
	"net"
	"os/exec"
	"time"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zenhack.net/go/sandstorm/grain"
)

func startZnc() {
	chkfatal(exec.Command("znc", "-f").Start())

	log.Println("Waiting for ZNC to start...")
	for {
		conn, err := net.Dial("tcp", zncAddr)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(time.Second / 10)
	}
	log.Println("ZNC is up.")
}

func main() {
	ctx := context.Background()

	netCaps := make(chan *ip_capnp.IpNetwork)
	configs := make(chan *ServerConfig)
	conns := make(chan net.Conn)

	go ipNetworkProxy(ctx, netCaps, configs, conns)

	writeConfig(&ZncConfig{
		ListenPort: zncPort,
		DialPort:   ipNetworkPort,
	})

	startZnc()

	api, err := grain.ConnectAPI(ctx, webui(ctx, netCaps, configs))
	chkfatal(err)
	api.StayAwake(ctx, nil).Handle()

	<-ctx.Done()
}
