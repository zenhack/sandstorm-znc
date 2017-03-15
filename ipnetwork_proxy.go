package main

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	util_capnp "zenhack.net/go/sandstorm/capnp/util"
	"zenhack.net/go/sandstorm/util"
)

func ipNetworkProxy(
	ctx context.Context,
	netCaps <-chan *ip_capnp.IpNetwork,
	configs <-chan *ServerConfig,
	conns chan net.Conn,
) {
	portCaps := make(chan *ip_capnp.TcpPort)
	go ipNetworkForwarder(ctx, portCaps, conns)
	go ipNetworkListener(conns)

	var (
		cap    *ip_capnp.IpNetwork
		config *ServerConfig
	)

	reconnect := func(cap *ip_capnp.IpNetwork, config *ServerConfig) {
		if cap == nil || config == nil {
			return
		}

		ipAddr := net.ParseIP(config.Host)
		var host ip_capnp.IpRemoteHost

		if ipAddr == nil {
			// not a valid ip; assume it's a hostname.
			host = cap.GetRemoteHostByName(
				ctx,
				func(p ip_capnp.IpNetwork_getRemoteHostByName_Params) error {
					p.SetAddress(config.Host)
					return nil
				}).Host()
		} else {
			ipAddr = ipAddr.To16()
			host = cap.GetRemoteHost(
				ctx,
				func(p ip_capnp.IpNetwork_getRemoteHost_Params) error {
					capnpAddr, err := p.NewAddress()
					if err != nil {
						return err
					}
					capnpAddr.SetUpper64(binary.BigEndian.Uint64(ipAddr[:8]))
					capnpAddr.SetUpper64(binary.BigEndian.Uint64(ipAddr[8:]))
					return nil
				}).Host()
		}

		port := host.GetTcpPort(
			ctx,
			func(p ip_capnp.IpRemoteHost_getTcpPort_Params) error {
				p.SetPortNum(config.Port)
				return nil
			}).Port()
		portCaps <- &port
	}

	for {
		select {
		case config = <-configs:
			if cap == nil {
				continue
			}
		case cap = <-netCaps:
			if config == nil {
				continue
			}
		}
		reconnect(cap, config)
	}
}

func ipNetworkForwarder(
	ctx context.Context,
	portCaps <-chan *ip_capnp.TcpPort,
	conns <-chan net.Conn,
) {
	var port *ip_capnp.TcpPort
	for {
		select {
		case port = <-portCaps:
		case conn := <-conns:
			if port == nil {
				log.Print("Got connection, but we don't have internet yet.")
				conn.Close()
				continue
			}
			serverConn := connect(ctx, port)
			go copyClose(serverConn, conn)
		}
	}
}

func connect(ctx context.Context, port *ip_capnp.TcpPort) net.Conn {
	clientConn, serverConn := net.Pipe()
	toServerBS := port.Connect(ctx, func(p ip_capnp.TcpPort_connect_Params) error {
		p.SetDownstream(util_capnp.ByteStream_ServerToClient(
			&util.WriteCloserByteStream{WC: serverConn},
		))
		return nil
	}).Upstream()
	go io.Copy(
		&util.ByteStreamWriteCloser{Ctx: ctx, Obj: toServerBS},
		serverConn,
	)
	return clientConn
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
