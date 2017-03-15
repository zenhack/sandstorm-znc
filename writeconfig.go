package main

import (
	// We import this under a different name, since we use
	// "html/template" as well. "text/template" is only used
	// for znc.conf.
	"os"
	txtTpl "text/template"
)

var (
	zncConfTpl = txtTpl.Must(txtTpl.ParseFiles("/opt/app/znc.conf"))
)

type ZncConfig struct {
	ListenPort, DialPort string
}

func writeConfig(cfg *ZncConfig) {
	configDir := os.Getenv("HOME") + "/.znc/configs"
	chkfatal(os.MkdirAll(configDir, 0700))
	file, err := os.Create(configDir + "/znc.conf")
	chkfatal(err)
	defer file.Close()
	chkfatal(zncConfTpl.Execute(file, cfg))
}
