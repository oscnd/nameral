package recordEndpoint

import (
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/nameral/type/payload"
	"go.scnd.dev/open/polygon/compat/response"
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
	if body.No == nil || body.Type == nil || body.Value == nil {
		return fiber.ErrBadRequest
	}

	// * get record to update
	rec := r.Store.GetRecordByNo(*body.No)
	if rec == nil {
		return fiber.ErrNotFound
	}

	// * update record in memory
	updated := r.Store.UpdateRecordByNo(*body.No, *body.Type, []string{*body.Value})
	if !updated {
		return fiber.ErrNotFound
	}

	err := r.Store.WriteLine(*body.No, *rec.Name, *body.Type, []string{*body.Value})
	if err != nil {
		return s.Error("failed to write line", err)
	}

	// * response
	return c.JSON(response.Success(s, true))
}
