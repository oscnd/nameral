package recordEndpoint

import (
	"go.scnd.dev/open/nameral/module/store"
	"go.scnn.net/base/scaff"
)

type Handler struct {
	Layer scaff.Layer
	Store *store.Store
}

func Handle(plg scaff.Scaff, s *store.Store) *Handler {
	return &Handler{
		Layer: plg.Layer("record", "endpoint"),
		Store: s,
	}
}
