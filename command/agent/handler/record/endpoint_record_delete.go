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

	r.Store.Mu.Lock()
	for fqdn, records := range r.Store.Records {
		filtered := records[:0]
		for _, rec := range records {
			if *rec.No != *body.No {
				filtered = append(filtered, rec)
			}
		}
		r.Store.Records[fqdn] = filtered
	}
	r.Store.Mu.Unlock()

	r.Store.Save()

	// * response
	return c.JSON(response.Success(s, true))
}
