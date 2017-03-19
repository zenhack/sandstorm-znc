package main

import (
	"context"
	"golang.org/x/net/websocket"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	grain_capnp "zenhack.net/go/sandstorm/capnp/grain"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zenhack.net/go/sandstorm/grain"
	"zenhack.net/go/sandstorm/websession"
	"zombiezen.com/go/capnproto2"
)

var (
	templates = template.Must(template.ParseGlob(appDir + "/templates/*"))
)

// Status information; passed to templates to report to clients.
type Status struct {
	HaveNetwork bool // whether we have an ipNetwork capability.
	Server      *ServerConfig
}

// Create the webui.
//
// netCaps and serverConfigs will be used to communicate changes to the ipNetwork
// cap and server config to the backend.
func webui(ctx context.Context,
	netCaps chan<- *ip_capnp.IpNetwork,
	serverConfigs chan<- *ServerConfig,
) websession.HandlerWebSession {

	badReq := func(w http.ResponseWriter) {
		w.WriteHeader(400)
		w.Write([]byte("Bad Request"))
	}

	coord := startCoordinator(ctx, serverConfigs, netCaps)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		templates.Lookup("index.html").Execute(w, Status{
			HaveNetwork: <-coord.getNetwork != nil,
			Server:      <-coord.getConfig,
		})
	})

	// Update the config:
	mux.HandleFunc("/config", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			badReq(w)
			return
		}

		port, err := strconv.ParseUint(req.FormValue("port"), 10, 16)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}
		if port == 0 {
			w.WriteHeader(400)
			w.Write([]byte("Port must be non-zero."))
			return
		}
		coord.setConfig <- &ServerConfig{
			Host: req.FormValue("irc-server"),
			Port: uint16(port),
			TLS:  req.FormValue("tls") == "on",
		}
		http.Redirect(w, req, "/", http.StatusSeeOther)
	})

	// Websocket connection, to be forwarded to ZNC:
	mux.Handle("/connect", websocket.Handler(func(wsConn *websocket.Conn) {
		zncConn, err := net.Dial("tcp", zncAddr)
		if err != nil {
			zncConn.Close()
			log.Printf("Error connecting to ZNC: %v", err)
			return
		}
		copyClose(zncConn, wsConn)
	}))

	// An IpNetwork capability; send it off to the backend so it can access
	// the internet:
	mux.HandleFunc("/ip-network-cap", func(w http.ResponseWriter, req *http.Request) {
		buf, err := ioutil.ReadAll(io.LimitReader(req.Body, 512))
		if err != nil {
			badReq(w)
			return
		}

		sessionCtx := w.(grain.HasSessionContext).GetSessionContext()
		results, err := sessionCtx.ClaimRequest(
			ctx,
			func(p grain_capnp.SessionContext_claimRequest_Params) error {
				p.SetRequestToken(string(buf))
				return nil
			}).Struct()
		if err != nil {
			badReq(w)
			return
		}
		cap, err := results.Cap()
		if err != nil {
			log.Printf("error claiming network cap: %v", err)
			return
		}
		coord.setNetwork <- &ip_capnp.IpNetwork{capnp.ToInterface(cap).Client()}
	})

	mux.Handle("/static/", http.FileServer(http.Dir(appDir)))

	return websession.FromHandler(ctx, mux)
}
