package dns

import "strings"

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
		for i, c := range z.clients {
			if c == cs {
				z.clients = append(z.clients[:i], z.clients[i+1:]...)
				break
			}
		}
	}
}
