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
	"zenhack.net/go/sandstorm/grain"
	"zenhack.net/go/sandstorm/websession"
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
// coord will be used to communicate changes to the ipNetwork
// cap and server config to the backend.
func webui(ctx context.Context, coord coordChans) websession.HandlerWebSession {

	badReq := func(w http.ResponseWriter) {
		w.WriteHeader(400)
		w.Write([]byte("Bad Request"))
	}

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
			// TODO: we should handle this case more gracefully; it can
			// easily happen if e.g. the grain has been shut down and is
			// woken by a request from the IRC client (rather than the
			// web ui). Possible improvements:
			//
			// * Easy: write a NOTICE message to the client, telling
			//   them what happened.
			// * Better: make sure ZNC is up before we start
			//   accepting connections. Reject websocket connections
			//   with a NOTICE if we can't start ZNC due to missing
			//   server config/network cap.
			//
			// The latter will take a bit more work, but is probably
			// worth it.
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
		coord.setNetwork <- cap
	})

	mux.Handle("/static/", http.FileServer(http.Dir(appDir)))

	return websession.FromHandler(ctx, mux)
}
