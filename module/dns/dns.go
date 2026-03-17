package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
	"go.scnd.dev/open/nameral/generate/proto"
)

type Module struct {
	mu       sync.RWMutex
	registry map[string][]*ClientStream // zone → ordered list of clients
}

func New(server *dns.Server, redis *redis.Client) *Module {
	m := &Module{
		registry: make(map[string][]*ClientStream),
	}
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		if len(r.Question) == 0 {
			replyCode(w, r, dns.RcodeServerFailure)
			return
		}

		q := r.Question[0]
		name := strings.ToLower(strings.TrimSuffix(q.Name, "."))
		qtype := dns.TypeToString[q.Qtype]
		ctx := context.Background()

		// Redis cache check
		key := fmt.Sprintf("dns:%s:%s", name, qtype)
		if cached, err := redis.Get(ctx, key).Result(); err == nil {
			var result proto.ResolveResult
			if json.Unmarshal([]byte(cached), &result) == nil {
				writeResponse(w, r, &result)
				return
			}
		}

		// Resolve via module registry
		result, err := m.Query(ctx, name, qtype)
		if err != nil {
			replyCode(w, r, dns.RcodeServerFailure)
			return
		}

		if result.Ttl == 0 {
			result.Ttl = uint32(60)
		}

		// Cache NOERROR with TTL > 0
		if result.Rcode == "NOERROR" && result.Ttl > 0 {
			if data, err := json.Marshal(result); err == nil {
				redis.Set(ctx, key, data, time.Duration(result.Ttl)*time.Second)
			}
		}

		writeResponse(w, r, result)
	})
	return m
}

func (r *Module) Register(cs *ClientStream, zones []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, zone := range zones {
		r.registry[zone] = append(r.registry[zone], cs)
	}
}

func (r *Module) Unregister(cs *ClientStream, zones []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, zone := range zones {
		list := r.registry[zone]
		for i, c := range list {
			if c == cs {
				r.registry[zone] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
}

func (r *Module) Query(ctx context.Context, name string, qtype string) (*proto.ResolveResult, error) {
	// Collect matching zones
	r.mu.RLock()
	var matchingZones []string
	for zone := range r.registry {
		if zone == "." || name == zone || strings.HasSuffix(name, "."+zone) {
			matchingZones = append(matchingZones, zone)
		}
	}
	r.mu.RUnlock()

	if len(matchingZones) == 0 {
		return &proto.ResolveResult{Rcode: "SERVFAIL"}, nil
	}

	// Sort by zone length descending (most specific first)
	sort.Slice(matchingZones, func(i, j int) bool {
		return len(matchingZones[i]) > len(matchingZones[j])
	})

	for _, zone := range matchingZones {
		r.mu.RLock()
		clients := make([]*ClientStream, len(r.registry[zone]))
		copy(clients, r.registry[zone])
		r.mu.RUnlock()

		for _, client := range clients {
			subdomain := name
			if zone != "." {
				if name == zone {
					subdomain = ""
				} else {
					subdomain = strings.TrimSuffix(name, "."+zone)
				}
			}

			no := client.no.Add(1)
			ch := make(chan *proto.ResolveResult, 1)
			client.pending.Store(no, ch)
			defer client.pending.Delete(no)

			query := &proto.ResolveQuery{
				No:        no,
				Type:      qtype,
				Zone:      zone,
				Subdomain: subdomain,
			}

			client.mu.Lock()
			err := client.Stream.Send(query)
			client.mu.Unlock()
			if err != nil {
				continue
			}

			select {
			case r := <-ch:
				if r.Rcode == "NOERROR" {
					return r, nil
				}
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled")
			}
		}
	}

	return &proto.ResolveResult{
		Rcode: "NXDOMAIN",
	}, nil
}
