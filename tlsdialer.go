package main

import (
	"crypto/tls"
	"net"
)

// This is the same as the Dialer interface from "golang.org/x/net/proxy".
// the Dial method has the same semantics as net.Dial from the standard
// library.
//
// We duplicate this here, rather than pull in an extra dependency from
// which we use so little.
type Dialer interface {
	Dial(network, addr string) (c net.Conn, err error)
}

// Dialer that speaks TLS over the `Base` Dialer, verifying the hostname
// it is passed.
type TLSDialer struct {
	Base Dialer
}

func (d *TLSDialer) Dial(network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	cfg := &tls.Config{
		ServerName: host,
	}
	if err != nil {
		return nil, err
	}
	conn, err := d.Base.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Client(conn, cfg)
	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}
	return tlsConn, nil
}
