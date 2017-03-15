package main

import (
	"context"
	"golang.org/x/net/websocket"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	grain_capnp "zenhack.net/go/sandstorm/capnp/grain"
	ip_capnp "zenhack.net/go/sandstorm/capnp/ip"
	"zenhack.net/go/sandstorm/grain"
	"zenhack.net/go/sandstorm/websession"
	"zombiezen.com/go/capnproto2"
)

func webui(ctx context.Context,
	netCaps chan<- *ip_capnp.IpNetwork,
	serverConfigs chan<- *ServerConfig) websession.HandlerWebSession {

	mux := http.NewServeMux()

	mux.Handle("/connect", websocket.Handler(func(wsConn *websocket.Conn) {
		zncConn, err := net.Dial("tcp", zncAddr)
		if err != nil {
			zncConn.Close()
			log.Printf("Error connecting to ZNC: %v", err)
			return
		}
		copyClose(zncConn, wsConn)
	}))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("Hello!"))
	})

	mux.HandleFunc("/ip-network-cap", func(w http.ResponseWriter, req *http.Request) {
		badReq := func() {
			w.WriteHeader(400)
			w.Write([]byte("Bad Request"))
		}
		buf, err := ioutil.ReadAll(io.LimitReader(req.Body, 512))
		if err != nil {
			badReq()
			return
		}

		sessionCtx := w.(grain.HasSessionContext).GetSessionContext()
		cap, err := sessionCtx.ClaimRequest(
			context.TODO(),
			func(p grain_capnp.SessionContext_claimRequest_Params) error {
				p.SetRequestToken(string(buf))
				return nil
			}).Cap().Struct()
		if err != nil {
			badReq()
			return
		}
		netCaps <- &ip_capnp.IpNetwork{capnp.ToInterface(cap).Client()}
	})

	return websession.FromHandler(ctx, mux)
}
