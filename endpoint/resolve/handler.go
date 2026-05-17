package resolveEndpoint

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.scnd.dev/open/nameral/common/config"
	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/nameral/module/dns"
	"go.scnn.net/base/scaff"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

type Handler struct {
	Layer       scaff.Layer
	Config      *config.Config
	Redis       *redis.Client
	Dns         *dns.Module
	ShutdownCtx context.Context
}

type ProtoHandler struct {
	proto.UnsafeResolverServer
	*Handler
}

func Handle(
	lc fx.Lifecycle,
	scf scaff.Scaff,
	registrar *grpc.Server,
	config *config.Config,
	rdb *redis.Client,
	module *dns.Module,
) *Handler {
	shutdownCtx, shutdown := context.WithCancel(context.Background())

	h := &Handler{
		Layer:       scf.Layer("state", "endpoint"),
		Config:      config,
		Redis:       rdb,
		Dns:         module,
		ShutdownCtx: shutdownCtx,
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			shutdown()
			return nil
		},
	})

	proto.RegisterResolverServer(registrar, &ProtoHandler{
		Handler: h,
	})

	return h
}
