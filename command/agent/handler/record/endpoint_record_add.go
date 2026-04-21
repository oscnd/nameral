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
		return s.Error("unable to parse body", err)
	}

	// * add record
	hash, err := r.Store.AddRecord(*body.Name, *body.Type, *body.Value)
	if err != nil {
		return s.Error("failed to add record", err)
	}

	// * response
	h := hash
	return c.JSON(response.Success(s, &payload.Record{
		Hash:  &h,
		Name:  body.Name,
		Type:  body.Type,
		Value: body.Value,
	}))
}
