package main

import (
	"context"
	"net"
	"os"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
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

type coordChans struct {
	getConfig  <-chan *ServerConfig
	setConfig  chan<- *ServerConfig
	getNetwork <-chan *ip_capnp.IpNetwork
	setNetwork chan<- *ip_capnp.IpNetwork
}

func startCoordinator(
	ctx context.Context,
	notifyConfig chan<- *ServerConfig,
	notifyNetwork chan<- *ip_capnp.IpNetwork,
) coordChans {

	getConfig := make(chan *ServerConfig)
	setConfig := make(chan *ServerConfig)
	getNetwork := make(chan *ip_capnp.IpNetwork)
	setNetwork := make(chan *ip_capnp.IpNetwork)

	go func() {
		var (
			config  *ServerConfig
			network *ip_capnp.IpNetwork
		)
		for {
			select {
			case getConfig <- config:
			case getNetwork <- network:
			case network = <-setNetwork:
				notifyNetwork <- network
			case config = <-setConfig:
				notifyConfig <- config
			}
		}
	}()

	return coordChans{
		getConfig:  getConfig,
		setConfig:  setConfig,
		getNetwork: getNetwork,
		setNetwork: setNetwork,
	}
}
