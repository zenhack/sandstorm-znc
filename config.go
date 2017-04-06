package main

import (
	"context"
	"log"
	"net"
	"os"
	grain_capnp "zenhack.net/go/sandstorm/capnp/grain"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zombiezen.com/go/capnproto2"
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
	ipNetworkCapFile = "/var/ipNetworkCap"
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
	setNetwork chan<- capnp.Pointer
}

// Starts a coordinator goroutine. This is used to coordinate the updating of
// the current ServerConfig and ipNetwork capability to use.
//
// Whenever the config is updated, it will be sent on notifyConfig, sand saved
// using saveServerConfig.
//
// Whenever the ipNetwork capability is updated, it will be sent on
// notifyNetwork, and saved via saveIpNetwork.
//
// The above sends must complete before the coordinator will service
// any more requests.
//
// The setApi paramter is a hack; it's used to get around a circular
// dependency. In particular:
//
// * the coordinator needs a SandstormApi to save the network cap.
// * the web ui needs a coordinator
// * the web ui is the bootstrap interface, which needs to be passed to
//   grain.ConnectAPI().
//
// To solve this, setApi is used to deliver the Api when it is available.
// Only one send may ever occur on the setApi channel, and the coordinator
// will not service requests until this occurs.
//
// XXX TODO: come up with a better solution to the above.
func startCoordinator(
	ctx context.Context,
	setApi <-chan grain_capnp.SandstormApi,
	notifyConfig chan<- *ServerConfig,
	notifyNetwork chan<- *ip_capnp.IpNetwork,
) coordChans {

	getConfig := make(chan *ServerConfig)
	setConfig := make(chan *ServerConfig)
	getNetwork := make(chan *ip_capnp.IpNetwork)
	setNetwork := make(chan capnp.Pointer)

	go func() {
		var (
			config  *ServerConfig
			network *ip_capnp.IpNetwork
		)
		api := <-setApi
		for {
			select {
			case getConfig <- config:
			case getNetwork <- network:
			case cap := <-setNetwork:
				// FIXME: We should only save this if we've never done so
				// before; it may be something we just restored.
				if err := saveIpNetwork(ctx, api, cap); err != nil {
					log.Println("Failed to persist ipNetwork cap.")
				}
				network = &ip_capnp.IpNetwork{capnp.ToInterface(cap).Client()}
				notifyNetwork <- network
			case config = <-setConfig:
				notifyConfig <- config
				if err := saveServerConfig(config); err != nil {
					log.Printf("Failed to write ServerConfig: %v", err)
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
