package endpoint

import (
	"go.scnd.dev/open/nameral/common/config"
	resolveHandler "go.scnd.dev/open/nameral/endpoint/resolve"
	"go.scnn.net/base/scaff"
)

func Bind(
	scf scaff.Scaff,
	config *config.Config,
	app *fiber.App,
	handler *resolveHandler.Handler,
) {
	api := app.Group("/api")
	_ = api
}
