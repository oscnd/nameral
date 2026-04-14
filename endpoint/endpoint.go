package endpoint

import (
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/nameral/common/config"
	resolveHandler "go.scnd.dev/open/nameral/endpoint/resolve"
	"go.scnd.dev/open/polygon"
)

func Bind(
	polygon polygon.Polygon,
	config *config.Config,
	app *fiber.App,
	handler *resolveHandler.Handler,
) {
	api := app.Group("/api")
	_ = api
}
