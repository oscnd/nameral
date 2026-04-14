package recordEndpoint

import (
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/nameral/type/payload"
	"go.scnd.dev/open/polygon/compat/response"
)

func (r *Handler) HandleDelete(c fiber.Ctx) error {
	// * span
	s, _ := r.Layer.With(c.Context())
	defer s.End()

	// * parse body
	body := new(payload.RecordDeleteBody)
	if err := c.Bind().JSON(body); err != nil {
		return s.Error("unable to parse body", err)
	}

	// * get record info before deleting
	rec := r.Store.GetRecordByNo(*body.No)
	if rec == nil {
		return s.Error("record not found", nil)
	}

	// * delete record line and reorder
	if err := r.Store.DeleteRecordByNo(*body.No); err != nil {
		return s.Error("failed to delete record", err)
	}

	// * response
	return c.JSON(response.Success(s, rec))
}
