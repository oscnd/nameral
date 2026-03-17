package recordEndpoint

import (
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/nameral/type/payload"
	"go.scnd.dev/open/polygon/compat/response"
)

func (r *Handler) HandleList(c fiber.Ctx) error {
	// * span
	s, _ := r.Layer.With(c.Context())
	defer s.End()

	r.Store.Mu.RLock()
	defer r.Store.Mu.RUnlock()

	var result []*payload.Record
	for _, records := range r.Store.Records {
		result = append(result, records...)
	}

	if result == nil {
		result = []*payload.Record{}
	}

	// * response
	return c.JSON(response.Success(s, result))
}
