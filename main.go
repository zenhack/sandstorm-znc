package main

import (
	"context"
	"os"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zenhack.net/go/sandstorm/grain"
	// We import this under a different name, since we use
	// "html/template" as well. "text/template" is only used
	// for znc.conf.
	txtTpl "text/template"
)

var (
	zncConfTpl = txtTpl.Must(txtTpl.ParseFiles("/opt/app/znc.conf"))
)

// Paratmeters for the znc configuration file.
type ZncConfig struct {
	ListenPort, DialPort string
}

// Write the znc config to the appropriate location.
func writeConfig(cfg *ZncConfig) {
	configDir := os.Getenv("HOME") + "/.znc/configs"
	chkfatal(os.MkdirAll(configDir, 0700))
	file, err := os.Create(configDir + "/znc.conf")
	chkfatal(err)
	defer file.Close()
	chkfatal(zncConfTpl.Execute(file, cfg))
}

func main() {
	ctx := context.Background()

	netCaps := make(chan *ip_capnp.IpNetwork)
	configs := make(chan *ServerConfig)

	go ipNetworkProxy(ctx, netCaps, configs)

	writeConfig(&ZncConfig{
		ListenPort: zncPort,
		DialPort:   ipNetworkPort,
	})

	api, err := grain.ConnectAPI(ctx, webui(ctx, netCaps, configs))
	chkfatal(err)
	api.StayAwake(ctx, nil).Handle()

	<-ctx.Done()
}
