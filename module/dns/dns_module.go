package dns

import (
	"sync"
	"sync/atomic"

	"go.scnn.net/base/scaff"
	"golang.org/x/sync/singleflight"
)

type Module struct {
	layer    scaff.Layer
	mutex    *sync.RWMutex
	no       *atomic.Uint64
	pending  *sync.Map           // uint64 → chan *proto.ResolveResult
	inflight *singleflight.Group // coalesces concurrent queries for the same key
	registry map[string]*Zone    // zone → zone entry
	redis    *redis.Client
}

type Zone struct {
	clients       []*ClientStream // ordered list of clients
	dnssecZoneKey *ZoneKey        // null if not a dnssec zone
}
