package dns

import (
	"sync"
	"sync/atomic"

	"go.scnd.dev/open/polygon"
)

type Module struct {
	layer    polygon.Layer
	mutex    *sync.RWMutex
	no       *atomic.Uint64
	pending  *sync.Map        // uint64 → chan *proto.ResolveResult
	stopCh   chan struct{}    // stop signal
	registry map[string]*Zone // zone → zone entry
}

type Zone struct {
	clients       []*ClientStream // ordered list of clients
	dnssecZoneKey *ZoneKey        // null if not a dnssec zone
}

func (r *Module) StopCh() <-chan struct{} {
	return r.stopCh
}
