package resolveEndpoint

import (
	"github.com/redis/go-redis/v9"
	"go.scnd.dev/open/nameral/common/config"
	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/nameral/module/dns"
	"go.scnd.dev/open/polygon"
	"google.golang.org/grpc"
)

type Handler struct {
	Layer  polygon.Layer
	Config *config.Config
	Redis  *redis.Client
	Dns    *dns.Module
}

type ProtoHandler struct {
	proto.UnsafeResolverServer
	*Handler
}

func Handle(
	polygon polygon.Polygon,
	registrar *grpc.Server,
	config *config.Config,
	rdb *redis.Client,
	module *dns.Module,
) *Handler {
	h := &Handler{
		Layer:  polygon.Layer("state", "endpoint"),
		Config: config,
		Redis:  rdb,
		Dns:    module,
	}

	proto.RegisterResolverServer(registrar, &ProtoHandler{
		Handler: h,
	})

	return h
}
