package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zenhack.net/go/sandstorm/ip"
)

// Start the ZNC daemon, and wait until it starts accepting connections
// before returning.
func startZnc() {
	cmd := exec.Command("znc", "-f")
	// Attach these so the sandstorm console shows output from ZNC.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	chkfatal(cmd.Start())

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

// Start ZNC, listen for connections from ZNC on `ipNetworkAddr`, and proxy
// them using sandstorm's ipNetwork.
//
// configs is used to receive updates to which endpoint to connect to.
//
// netCaps is used to receive the lastest ipNetwork capability that should
// be used to make the connection.
//
// ZNC will not be started until at least one receive has succeded on both
// netCaps and configs.
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

// Accept connections from ipNetworkAddr, and send them on 'conns'.
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
