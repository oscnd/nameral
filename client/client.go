package client

import (
	"context"
	"sync"

	"go.scnd.dev/open/nameral"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func New(config *Config) (nameral.Nameral, error) {
	conn, err := grpc.NewClient(*config.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Namera{
		config:   config,
		conn:     conn,
		handlers: &sync.Map{},
		cancel:   cancel,
	}
	go r.stream(ctx)
	return r, nil
}
