package dns

import (
	"strings"

	"go.scnd.dev/open/polygon/utility/value"
)

func (r *Module) Register(cs *ClientStream, zones []string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, zone := range zones {
		if r.registry[zone] == nil {
			var zk *ZoneKey
			for dnssecZone, z := range r.registry {
				if z.dnssecZoneKey != nil && (zone == dnssecZone || strings.HasSuffix(zone, dnssecZone)) {
					zk = z.dnssecZoneKey
					break
				}
			}
			r.registry[zone] = &Zone{dnssecZoneKey: zk}
		}
		r.registry[zone].clients = append(r.registry[zone].clients, cs)
	}
}

func (r *Module) Unregister(cs *ClientStream, zones []string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, zone := range zones {
		z := r.registry[zone]
		if z == nil {
			continue
		}
		if i := value.Index(z.clients, cs); i >= 0 {
			z.clients = value.RemoveIndex(z.clients, i)
		}
	}
}
