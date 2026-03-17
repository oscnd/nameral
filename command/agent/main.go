package main

import (
	"go.scnd.dev/open/polygon/compat/common"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			common.Config[Config],
		),
		fx.Invoke(
			invoke,
		),
	).Run()
}

func invoke(config *Config) {

}
