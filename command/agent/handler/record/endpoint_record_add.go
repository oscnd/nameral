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
	if body.Name == nil || body.Type == nil || len(body.Values) == 0 {
		return fiber.ErrBadRequest
	}

	typ := *body.Type
	name := *body.Name

	// * convert []*string to []string
	vals := make([]string, len(body.Values))
	for i, v := range body.Values {
		vals[i] = *v
	}

	// * add record and get line number
	no, err := r.Store.AddRecord(name, typ, vals)
	if err != nil {
		return s.Error("failed to add record", err)
	}

	// * response
	return c.JSON(response.Success(s, &payload.Record{
		No:     &no,
		Name:   &name,
		Type:   &typ,
		Values: body.Values,
	}))
}
