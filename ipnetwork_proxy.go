package main

import (
	"context"
	"log"
	"net"
	"os/exec"
	"strconv"
	"time"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zenhack.net/go/sandstorm/ip"
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

func ipNetworkProxy(
	ctx context.Context,
	netCaps <-chan *ip_capnp.IpNetwork,
	configs <-chan *ServerConfig,
) {

	var (
		config *ServerConfig
		cap    *ip_capnp.IpNetwork
	)

	for cap == nil || config == nil {
		select {
		case cap = <-netCaps:
		case config = <-configs:
		}
	}

	conns := make(chan net.Conn)
	go ipNetworkListener(conns)

	startZnc()

	for {
		select {
		case config = <-configs:
		case cap = <-netCaps:
		case zncConn := <-conns:
			log.Printf("Got connection from znc")
			dialer := &ip.IpNetworkDialer{
				Ctx:       ctx,
				IpNetwork: *cap,
			}
			serverConn, err := dialer.Dial(
				"tcp",
				net.JoinHostPort(
					config.Host,
					strconv.Itoa(int(config.Port)),
				),
			)
			if err != nil {
				log.Printf("error connecting to irc server: %v", err)
				zncConn.Close()
				continue
			}
			go copyClose(serverConn, zncConn)
		}
	}
}

func ipNetworkListener(conns chan<- net.Conn) {
	l, err := net.Listen("tcp", ipNetworkAddr)
	chkfatal(err)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("Error in Accept(): %v")
		}
		conns <- conn
	}
}
