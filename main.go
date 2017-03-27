package main

import (
	"context"
	"encoding/json"
	"log"
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

// Check if a file exists.
func exists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
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

	// We use the file ${HOME}/HAVE_CONFIG to indicate whether we've
	// written the config in the past; it will only be absent on the first
	// bringup. We don't want to override it if so, since the user may
	// have asked ZNC to apply changes.
	//
	// We could just check for the presense of the config file itself,
	// but that has two problems:
	//
	// 1. For users who installed alpha 1, we want this to upgrade the
	//    (non-user-modifiable) config that that version created.
	// 2. In the even that the system dies in the middle of writing the
	//    config, checking for the config woud permanently corrupt the
	//    grain, since we'd never remove/overwrite the partial config.
	haveCfgPath := os.Getenv("HOME") + "/HAVE_CONFIG"
	haveCfg, err := exists(haveCfgPath)
	chkfatal(err)
	if !haveCfg {
		writeConfig(&ZncConfig{
			ListenPort: zncPort,
			DialPort:   ipNetworkPort,
		})
		file, err := os.Create(haveCfgPath)
		chkfatal(err)
		file.Close()
	}

	coord := startCoordinator(ctx, configs, netCaps)

	// If we have an existing server config, load it.
	file, err := os.Open(serverConfigPath)
	if err == nil {
		config := &ServerConfig{}
		err = json.NewDecoder(file).Decode(config)
		file.Close()
		if err == nil {
			log.Printf("Loaded saved config from %q.", serverConfigPath)
			coord.setConfig <- config
		} else {
			log.Printf("Failed decoding ServerConfig from %q: %v",
				serverConfigPath, err)
		}
	} else {
		log.Printf("Failed to load ServerConfig from %q: %v. Note that "+
			"this is normal on first startup.", serverConfigPath, err)
	}

	api, err := grain.ConnectAPI(ctx, webui(ctx, coord))
	chkfatal(err)
	api.StayAwake(ctx, nil).Handle()

	<-ctx.Done()
}
