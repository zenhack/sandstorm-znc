package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
)

const (
	lo = "127.0.0.1"
)

var (
	// These are set in sandstorm-pkgdef.capnp.
	zncPort       = os.Getenv("SANDSTORM_ZNC_PORT")
	ipNetworkPort = os.Getenv("SANDSTORM_IP_NETWORK_PORT")
	appDir        = os.Getenv("SANDSTORM_APP_DIR")

	zncAddr       = net.JoinHostPort(lo, zncPort)
	ipNetworkAddr = net.JoinHostPort(lo, ipNetworkPort)

	serverConfigPath = "/var/ServerConfig.json"
)

// A ServerConfig specifies a server to connect to.
type ServerConfig struct {
	Host string // Hostname of the server
	Port uint16 // TCP port number
	TLS  bool   // Whether to connect via TLS
}

// coordChans allows communication with the coordinator; see startCoodinator.
type coordChans struct {
	getConfig  <-chan *ServerConfig
	setConfig  chan<- *ServerConfig
	getNetwork <-chan *ip_capnp.IpNetwork
	setNetwork chan<- *ip_capnp.IpNetwork
}

// Starts a coordinator goroutine. This is used to coordinate the updating of
// the current ServerConfig and ipNetwork capability to use.
//
// Whenever the config is updated, it will be sent on notifyConfig.
//
// Whenever the ipNetwork capability is updated, it will be sent on
// notifyNetwork.
//
// The above sends must complete before the coordinator will service
// any more requests.
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
				file, err := os.Create(serverConfigPath)
				if err != nil {
					log.Printf("Failed to open %q for writing: %v",
						serverConfigPath, err)
					continue
				}
				err = json.NewEncoder(file).Encode(config)
				file.Close()
				if err != nil {
					log.Printf("Failed to write ServerConfig to %q: %v",
						serverConfigPath, err)
					continue
				}
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
