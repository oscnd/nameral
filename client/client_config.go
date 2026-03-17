package client

import "crypto/tls"

type Config struct {
	Address *string
	Secret  *string
	Tls     *tls.Config // nil = insecure
}
