package recordEndpoint

import (
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/nameral/type/payload"
	"go.scnd.dev/open/polygon/compat/response"
)

func (r *Handler) HandleAdd(c fiber.Ctx) error {
	// * span
	s, _ := r.Layer.With(c.Context())
	defer s.End()

	// * parse body
	body := new(payload.RecordAddBody)
	if err := c.Bind().JSON(body); err != nil {
		return fiber.ErrBadRequest
	}
	if body.Name == nil || body.Type == nil || body.Value == nil {
		return fiber.ErrBadRequest
	}

	typ := *body.Type
	val := *body.Value
	name := *body.Name

	// * add record and get line number
	no, err := r.Store.AddRecord(name, typ, []string{val})
	if err != nil {
		return s.Error("failed to add record", err)
	}

	// * response
	return c.JSON(response.Success(s, no))
}
