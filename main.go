package main

import (
	"go.scnd.dev/open/nameral/common/config"
	"go.scnd.dev/open/nameral/common/dns"
	"go.scnd.dev/open/nameral/common/grpc"
	"go.scnd.dev/open/nameral/endpoint"
	resolveEndpoint "go.scnd.dev/open/nameral/endpoint/resolve"
	dnsModule "go.scnd.dev/open/nameral/module/dns"
	"go.scnd.dev/open/polygon/compat/common"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			fx.Annotate(
				common.Config[config.Config],
				fx.As(new(common.FiberConfig)),
				fx.As(new(common.RedisConfig)),
				fx.As(new(common.PolygonConfig)),
			),
			common.Config[config.Config],
			common.Fiber,
			common.Redis,
			common.Polygon,
			grpc.Init,
			dns.Init,
			dnsModule.New,
			resolveEndpoint.Handle,
		),
		fx.Invoke(
			endpoint.Bind,
		),
	).Run()
}
