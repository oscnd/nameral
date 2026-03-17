package handler

import (
	"github.com/gofiber/fiber/v3"
	recordEndpoint "go.scnd.dev/open/nameral/command/agent/handler/record"
)

type Config interface {
	GetRecordKey() *string
}

func Bind(config Config, app *fiber.App, handler *recordEndpoint.Handler) {
	if app == nil {
		return
	}

	app.Use(func(c fiber.Ctx) error {
		key := config.GetRecordKey()
		if key == nil || c.Get("Authorization") != "Bearer "+*key {
			return fiber.ErrUnauthorized
		}
		return c.Next()
	})

	api := app.Group("/api")
	record := api.Group("/record")
	record.Post("/list", handler.HandleList)
	record.Post("/add", handler.HandleAdd)
	record.Post("/set", handler.HandleSet)
	record.Post("/delete", handler.HandleDelete)
}
