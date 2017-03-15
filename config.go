package main

import (
	"net"
	"os"
	"sync"
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

// Atomic cell for the current ServerConfig.
type configCell struct {
	sync.RWMutex
	value *ServerConfig
}

func (c *configCell) Get() *ServerConfig {
	c.RLock()
	defer c.RUnlock()
	return c.value
}

func (c *configCell) Set(cfg *ServerConfig) {
	c.Lock()
	defer c.Unlock()
	c.value = cfg
}
