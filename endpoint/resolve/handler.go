package resolveEndpoint

import (
	"go.scnd.dev/open/nameral/common/config"
	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/nameral/module/dns"
	"go.scnn.net/base/scaff"
	"google.golang.org/grpc"
)

type Handler struct {
	Layer  scaff.Layer
	Config *config.Config
	Redis  *redis.Client
	Dns    *dns.Module
}

type ProtoHandler struct {
	proto.UnsafeResolverServer
	*Handler
}

func Handle(
	scf scaff.Scaff,
	registrar *grpc.Server,
	config *config.Config,
	rdb *redis.Client,
	module *dns.Module,
) *Handler {
	h := &Handler{
		Layer:  scf.Layer("state", "endpoint"),
		Config: config,
		Redis:  rdb,
		Dns:    module,
	}

	proto.RegisterResolverServer(registrar, &ProtoHandler{
		Handler: h,
	})

	return h
}
