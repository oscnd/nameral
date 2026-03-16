package main

import (
	"embed"

	"go.scnd.dev/open/nameral/common/config"
	"go.scnd.dev/open/polygon/compat/common"
	"go.scnd.dev/open/polygon/compat/predefine"
	"go.uber.org/fx"
)

//go:embed _database/migration/*.sql
var migration embed.FS

//go:embed .local/dist/*
var frontend embed.FS

func main() {
	fx.New(
		fx.Provide(
			func() predefine.FrontendFS {
				return frontend
			},
			func() predefine.MigrationFS {
				return migration
			},
			fx.Annotate(
				common.Config[config.Config],
				fx.As(new(common.FiberConfig)),
				fx.As(new(common.RedisConfig)),
				fx.As(new(common.PolygonConfig)),
			),
			common.Config[config.Config],
			common.Fiber,
			common.Polygon,
		),
		fx.Invoke(),
	).Run()
}
