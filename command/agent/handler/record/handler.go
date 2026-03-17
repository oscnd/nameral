package recordEndpoint

import (
	"go.scnd.dev/open/nameral/module/store"
	"go.scnd.dev/open/polygon"
)

type Handler struct {
	Layer polygon.Layer
	Store *store.Store
}

func Handle(plg polygon.Polygon, s *store.Store) *Handler {
	return &Handler{
		Layer: plg.Layer("record", "endpoint"),
		Store: s,
	}
}
