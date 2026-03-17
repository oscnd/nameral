package nameral

import "go.scnd.dev/open/nameral/type/model"

type Nameral interface {
	Handle(zone string, handle func(*model.HandleQuery) (*model.HandleResponse, error))
	Flush(zone string) error
	Close() error
}
