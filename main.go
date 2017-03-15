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

const (
	zncAddr       = "127.0.0.1:8000"
	ipNetworkAddr = "127.0.0.1:6667"
)

// A ServerConfig specifies a server to connect to.
type ServerConfig struct {
	Host string // Hostname of the server
	Port uint16 // TCP port number
	TLS  bool   // Whether to connect via TLS
}

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
	startZnc()

	api, err := grain.ConnectAPI(ctx, webui(ctx, netCaps, configs))
	chkfatal(err)
	api.StayAwake(ctx, nil).Handle()

	<-ctx.Done()
}
