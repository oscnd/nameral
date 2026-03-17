package recordEndpoint

import (
	"github.com/gofiber/fiber/v3"
	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/type/payload"
	"go.scnd.dev/open/polygon/compat/response"
)

func (r *Handler) HandleAdd(c fiber.Ctx) error {
	// * span
	s, _ := r.Layer.With(c.Context())
	defer s.End()

	// * parse body
	body := new(payload.RecordSetBody)
	if err := c.Bind().JSON(body); err != nil {
		return fiber.ErrBadRequest
	}
	if body.Name == nil || body.Type == nil || body.Value == nil {
		return fiber.ErrBadRequest
	}

	fqdn := dns.Fqdn(*body.Name)
	typ := *body.Type
	val := *body.Value

	r.Store.Mu.Lock()

	no := r.Store.NextNo
	r.Store.NextNo++
	name := *body.Name
	r.Store.Records[fqdn] = append(r.Store.Records[fqdn], &payload.Record{
		No:     &no,
		Name:   &name,
		Type:   &typ,
		Values: []*string{&val},
	})

	r.Store.Mu.Unlock()

	r.Store.Save()

	// * response
	return c.JSON(response.Success(s, true))
}
