package handler

import (
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/nameral/common/config"
	"go.scnd.dev/open/nameral/handler/resolve"
	"go.scnd.dev/open/polygon/compat/predefine"
)

func Bind(
	frontend predefine.FrontendFS,
	config *config.Config,
	app *fiber.App,
	handler resolveHandler.Handler,
) {
	api := app.Group("/api")
	_ = api
}
