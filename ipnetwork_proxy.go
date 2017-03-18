package main

import (
	"context"
	"log"
	"net"
	"strconv"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zenhack.net/go/sandstorm/ip"
)

func ipNetworkProxy(
	ctx context.Context,
	netCaps <-chan *ip_capnp.IpNetwork,
	configs <-chan *ServerConfig,
) {

	var (
		config *ServerConfig
		cap    *ip_capnp.IpNetwork
	)

	conns := make(chan net.Conn)
	go ipNetworkListener(conns)

	for {
		log.Printf("Config: %v, Cap: %v", config, cap)
		select {
		case config = <-configs:
		case cap = <-netCaps:
		case zncConn := <-conns:
			if config == nil {
				log.Print("IpNetwork Proxy got a connection " +
					"from ZNC, but we don't have our config yet.")
				zncConn.Close()
				continue
			}
			if cap == nil {
				log.Print("IpNetwork Proxy got a connection " +
					"from ZNC, but we don't have internet access yet.")
				zncConn.Close()
			}
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
