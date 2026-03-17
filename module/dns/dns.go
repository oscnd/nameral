package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bsthun/gut"
	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
	"go.scnd.dev/open/nameral/common/config"
	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/polygon"
	"go.uber.org/fx"
)

func New(lifecycle fx.Lifecycle, polygon polygon.Polygon, cfg *config.Config, server *dns.Server, redis *redis.Client) *Module {
	m := &Module{
		layer:    polygon.Layer("dns", "module"),
		mutex:    new(sync.RWMutex),
		no:       new(atomic.Uint64),
		pending:  new(sync.Map),
		stopCh:   make(chan struct{}),
		registry: make(map[string]*Zone),
	}

	// * span
	s, _ := m.layer.With(context.TODO())
	defer s.End()

	if cfg.DnssecPath != nil && len(cfg.DnssecZones) > 0 {
		for _, zone := range cfg.DnssecZones {
			zk, err := loadZoneKey(*cfg.DnssecPath, *zone)
			if err != nil {
				gut.Fatal(fmt.Sprintf("dnssec loading failed for zone %s", *zone), err)
			}
			m.registry[*zone] = &Zone{dnssecZoneKey: zk}
			fmt.Printf("[dnssec] %s\n", zk.Ds())
		}
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return nil
		},
		OnStop: func(context.Context) error {
			close(m.stopCh)
			m.mutex.Lock()
			m.registry = make(map[string]*Zone)
			m.mutex.Unlock()
			return nil
		},
	})
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		if len(r.Question) == 0 {
			replyCode(w, r, dns.RcodeServerFailure)
			return
		}

		q := r.Question[0]
		name := strings.ToLower(strings.TrimSuffix(q.Name, "."))
		qtype := dns.TypeToString[q.Qtype]
		ctx := context.Background()

		do := r.IsEdns0() != nil && r.IsEdns0().Do()
		zk := m.dnssecMatchZone(name)

		// Handle DNSKEY queries locally for DNSSEC zones
		if q.Qtype == dns.TypeDNSKEY && zk != nil {
			msg := newMessage(r)
			msg.Authoritative = true
			msg.Answer = append(msg.Answer, dns.Copy(zk.dnsKey))
			if do {
				m.dnssecSign(msg, msg.Answer[:1], zk)
			}
			_ = w.WriteMsg(msg)
			return
		}

		// Redis cache check
		key := fmt.Sprintf("dns:%s:%s", name, qtype)
		if cached, err := redis.Get(ctx, key).Result(); err == nil {
			var result proto.ResolveResult
			if json.Unmarshal([]byte(cached), &result) == nil {
				msg := buildResponse(r, &result)
				if do && zk != nil && len(msg.Answer) > 0 {
					m.dnssecSign(msg, msg.Answer, zk)
				}
				w.WriteMsg(msg)
				return
			}
		}

		// Resolve via module registry
		result, err := m.Query(ctx, name, qtype)
		if err != nil {
			replyCode(w, r, dns.RcodeServerFailure)
			return
		}

		if result.Ttl < 60 {
			result.Ttl = uint32(60)
		}

		// Cache NOERROR with TTL > 0
		if result.Rcode == "NOERROR" && result.Ttl > 0 {
			if data, err := json.Marshal(result); err == nil {
				redis.Set(ctx, key, data, time.Duration(result.Ttl)*time.Second)
			}
		}

		msg := buildResponse(r, result)
		if do && zk != nil && len(msg.Answer) > 0 {
			m.dnssecSign(msg, msg.Answer, zk)
		}
		_ = w.WriteMsg(msg)
	})
	return m
}
