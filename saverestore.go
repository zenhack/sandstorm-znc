package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	grain_capnp "zenhack.net/go/sandstorm/capnp/grain"
	"zombiezen.com/go/capnproto2"
)

func loadServerConfig() (*ServerConfig, error) {
	// If we have an existing server config, load it.
	file, err := os.Open(serverConfigPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	config := &ServerConfig{}
	err = json.NewDecoder(file).Decode(config)
	if err != nil {
		log.Printf("Failed decoding ServerConfig from %q: %v",
			serverConfigPath, err)
	}
	return config, nil
}

func saveServerConfig(config *ServerConfig) error {
	file, err := os.Create(serverConfigPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(config)
}

func loadIpNetwork(ctx context.Context, api grain_capnp.SandstormApi) (capnp.Pointer, error) {
	token, err := ioutil.ReadFile(ipNetworkCapFile)
	if err != nil {
		return nil, err
	}
	capability, err := api.Restore(ctx,
		func(p grain_capnp.SandstormApi_restore_Params) error {
			p.SetToken(token)
			return nil
		}).Cap().Struct()
	if err != nil {
		return nil, err
	}
	return capability, nil
}

func saveIpNetwork(ctx context.Context, api grain_capnp.SandstormApi, ipNetworkCap capnp.Pointer) error {
	results, err := api.Save(
		ctx,
		func(p grain_capnp.SandstormApi_save_Params) error {
			p.SetCap(ipNetworkCap)
			label, err := p.NewLabel()
			if err != nil {
				return err
			}
			label.SetDefaultText("To access the IRC network")
			return nil
		},
	).Struct()
	if err != nil {
		return err
	}
	token, err := results.Token()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(ipNetworkCapFile, token, 0600)
}
