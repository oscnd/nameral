package dns

import (
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
	"go.scnd.dev/open/polygon"
)

type Module struct {
	layer    polygon.Layer
	mutex    *sync.RWMutex
	no       *atomic.Uint64
	pending  *sync.Map        // uint64 → chan *proto.ResolveResult
	inflight *sync.Map        // string (cache key) → chan *model.ResolveResult
	stopCh   chan struct{}    // stop signal
	registry map[string]*Zone // zone → zone entry
	redis    *redis.Client
}

type Zone struct {
	clients       []*ClientStream // ordered list of clients
	dnssecZoneKey *ZoneKey        // null if not a dnssec zone
}

func (r *Module) StopCh() <-chan struct{} {
	return r.stopCh
}
