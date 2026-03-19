package recordEndpoint

import (
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/nameral/type/payload"
	"go.scnd.dev/open/polygon/compat/response"
	"go.scnd.dev/open/polygon/utility/value"
)

func (r *Handler) HandleSet(c fiber.Ctx) error {
	// * span
	s, _ := r.Layer.With(c.Context())
	defer s.End()

	// * parse body
	body := new(payload.RecordSetBody)
	if err := c.Bind().JSON(body); err != nil {
		return fiber.ErrBadRequest
	}
	if body.No == nil || body.Type == nil || len(body.Value) == 0 {
		return fiber.ErrBadRequest
	}

	// * get record to update
	rec := r.Store.GetRecordByNo(*body.No)
	if rec == nil {
		return fiber.ErrNotFound
	}

	// * convert to value
	vals := value.ValSlice(body.Value)

	// * update record in memory
	updated := r.Store.UpdateRecordByNo(*body.No, *body.Type, vals)
	if !updated {
		return fiber.ErrNotFound
	}

	err := r.Store.WriteLine(*body.No, *rec.Name, *body.Type, vals)
	if err != nil {
		return s.Error("failed to write line", err)
	}

	// * response
	return c.JSON(response.Success(s, &payload.Record{
		No:     rec.No,
		Name:   rec.Name,
		Type:   body.Type,
		Values: body.Value,
	}))
}
