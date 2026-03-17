package handler

import (
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/nameral/common/config"
	resolveHandler "go.scnd.dev/open/nameral/handler/resolve"
)

func Bind(
	config *config.Config,
	app *fiber.App,
	handler *resolveHandler.Handler,
) {
	api := app.Group("/api")
	_ = api
}
