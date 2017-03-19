package main

import (
	"context"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zenhack.net/go/sandstorm/grain"
)

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
