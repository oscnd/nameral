package client

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.scnd.dev/open/nameral/client/nameral"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func New(config *Config) (nameral.Nameral, error) {
	var creds credentials.TransportCredentials
	if config.Tls != nil {
		creds = credentials.NewTLS(config.Tls)
	} else {
		creds = insecure.NewCredentials()
	}

	conns := make([]*grpc.ClientConn, 0, len(config.Addresses))
	for _, addr := range config.Addresses {
		target := *addr
		if !strings.Contains(target, "///") {
			target = "dns:///" + target
		}
		conn, err := grpc.NewClient(
			target,
			grpc.WithTransportCredentials(creds),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                2 * time.Second,
				Timeout:             1 * time.Second,
				PermitWithoutStream: true,
			}),
		)
		if err != nil {
			for _, c := range conns {
				c.Close()
			}
			return nil, err
		}
		conns = append(conns, conn)
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Namera{
		config:   config,
		conns:    conns,
		handlers: &sync.Map{},
		cancel:   cancel,
	}
	for _, conn := range conns {
		go r.stream(ctx, conn)
	}
	return r, nil
}
