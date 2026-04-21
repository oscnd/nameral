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
		return s.Error("unable to parse body", err)
	}

	// * get record to update
	rec := r.Store.GetRecordByHash(*body.Hash)
	if rec == nil {
		return s.Error("record not found", nil)
	}

	// * update record
	updated := r.Store.UpdateRecordByHash(*body.Hash, *body.Type, *body.Value)
	if !updated {
		return s.Error("failed to update record", nil)
	}

	// * response
	return c.JSON(response.Success(s, &payload.Record{
		Hash:  rec.Hash,
		Name:  rec.Name,
		Type:  body.Type,
		Value: body.Value,
	}))
}
