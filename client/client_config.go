package client

import "crypto/tls"

type Config struct {
	Addresses   []*string
	Secret      *string
	Tls         *tls.Config // nil = insecure
	OnConnect   func(addr string)
	OnReconnect func(addr string)
	OnResolve   func(addr string, no uint64, typ string, zone string, subdomain string)
}
