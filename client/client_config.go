package client

import "crypto/tls"

type Config struct {
	Addresses []*string
	Secret    *string
	Tls       *tls.Config // nil = insecure
}
