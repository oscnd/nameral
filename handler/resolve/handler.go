package resolveEndpoint

import (
	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/polygon"
	"google.golang.org/grpc"
)

type Handler struct {
	proto.UnsafeResolverServer
	Layer polygon.Layer
}

func Handle(
	polygon polygon.Polygon,
	registrar *grpc.Server,
) *Handler {
	h := &Handler{
		Layer: polygon.Layer("state", "endpoint"),
	}
	proto.RegisterResolverServer(registrar, h)
	return h
}
