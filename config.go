package main

import (
	"context"
	"net"
	"os"
)

const (
	lo = "127.0.0.1"
)

var (
	zncPort       = os.Getenv("SANDSTORM_ZNC_PORT")
	ipNetworkPort = os.Getenv("SANDSTORM_IP_NETWORK_PORT")
	appDir        = os.Getenv("SANDSTORM_APP_DIR")

	zncAddr       = net.JoinHostPort(lo, zncPort)
	ipNetworkAddr = net.JoinHostPort(lo, ipNetworkPort)
)

// A ServerConfig specifies a server to connect to.
type ServerConfig struct {
	Host string // Hostname of the server
	Port uint16 // TCP port number
	TLS  bool   // Whether to connect via TLS
}

type configProc struct {
	get <-chan *ServerConfig
	set chan<- *ServerConfig
}

func newConfigProc(
	ctx context.Context,
	init *ServerConfig,
	notify chan<- *ServerConfig,
) *configProc {
	get := make(chan *ServerConfig)
	set := make(chan *ServerConfig)
	go func() {
		current := init
		for {
			select {
			case <-ctx.Done():
				return
			case get <- current:
			case current = <-set:
				notify <- current
			}
		}
	}()
	return &configProc{get: get, set: set}
}
