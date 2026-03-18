package dns

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/nameral/type/model"
)

func (r *Module) Query(ctx context.Context, name string, qtype string) (*proto.ResolveResult, error) {
	// Collect matching zones
	r.mutex.RLock()
	var matchingZones []string
	for zone := range r.registry {
		if zone == "." || name == zone || strings.HasSuffix(name, "."+zone) {
			matchingZones = append(matchingZones, zone)
		}
	}
	r.mutex.RUnlock()

	if len(matchingZones) == 0 {
		return &proto.ResolveResult{Rcode: string(model.RcodeSERVFAIL)}, nil
	}

	// Sort by zone length descending (most specific first)
	sort.Slice(matchingZones, func(i, j int) bool {
		return len(matchingZones[i]) > len(matchingZones[j])
	})

	for _, zone := range matchingZones {
		r.mutex.RLock()
		clients := make([]*ClientStream, len(r.registry[zone].clients))
		copy(clients, r.registry[zone].clients)
		r.mutex.RUnlock()

		for _, client := range clients {
			subdomain := name
			if zone != "." {
				if name == zone {
					subdomain = ""
				} else {
					subdomain = strings.TrimSuffix(name, "."+zone)
				}
			}

			no := r.no.Add(1)
			ch := make(chan *proto.ResolveResult, 1)
			r.pending.Store(no, ch)
			defer r.pending.Delete(no)

			query := &proto.ResolveQuery{
				No:        no,
				Type:      qtype,
				Zone:      zone,
				Subdomain: subdomain,
			}

			client.Mutex.Lock()
			err := client.Stream.Send(query)
			client.Mutex.Unlock()
			if err != nil {
				continue
			}

			select {
			case res := <-ch:
				if res.Rcode == string(model.RcodeNOERROR) {
					return res, nil
				}
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled")
			}
		}
	}

	return &proto.ResolveResult{
		Rcode: string(model.RcodeNXDOMAIN),
	}, nil
}
