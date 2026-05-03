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
	"go.scnd.dev/open/nameral/type/model"
	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/utility/form"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
)

func New(lifecycle fx.Lifecycle, polygon polygon.Polygon, cfg *config.Config, server *dns.Server, redis *redis.Client) *Module {
	m := &Module{
		layer:    polygon.Layer("dns", "module"),
		mutex:    new(sync.RWMutex),
		no:       new(atomic.Uint64),
		pending:  new(sync.Map),
		inflight: new(singleflight.Group),
		registry: make(map[string]*Zone),
		redis:    redis,
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

		ctx := context.Background()
		q := r.Question[0]
		name := strings.ToLower(strings.TrimSuffix(q.Name, "."))
		qtype := dns.TypeToString[q.Qtype]

		requestSalt := *form.Random(form.RandomMixedCaseAlphaNum, 8)
		println("salt", requestSalt, "for", name, qtype, "from", w.RemoteAddr().String())

		do := r.IsEdns0() != nil && r.IsEdns0().Do()
		zk := m.DnssecMatchZone(name)

		// * handle dnskey queries locally for dnssec zones
		if q.Qtype == dns.TypeDNSKEY && zk != nil {
			msg := newMessage(r)
			msg.Authoritative = true
			msg.Answer = append(msg.Answer, dns.Copy(zk.dnsKey))
			if do {
				m.DnssecSign(&msg.Answer, msg.Answer[:1])
			}
			_ = w.WriteMsg(msg)
			return
		}

		// * redis cache check
		key := fmt.Sprintf("dns:%s:%s", name, qtype)
		if cached, err := redis.Get(ctx, key).Result(); err == nil {
			result := new(model.ResolveResult)
			if json.Unmarshal([]byte(cached), result) != nil {
				replyCode(w, r, dns.RcodeServerFailure)
				return
			}

			// * if resolution staled, return immediately but re-query in background (once)
			if result.ResolvedAt != nil && time.Since(*result.ResolvedAt) > 30*time.Second {
				m.ResolveResponseSend(w, r, result, do, zk)
				m.inflight.DoChan(key, func() (any, error) {
					return m.ResolveQuery(context.Background(), key, name, qtype)
				})
				return
			}

			m.ResolveResponseSend(w, r, result, do, zk)
			return
		}

		// * coalesce concurrent requests for the same key; singleflight broadcasts to all callers
		v, err, _ := m.inflight.Do(key, func() (any, error) {
			return m.ResolveQuery(ctx, key, name, qtype)
		})
		if err != nil {
			replyCode(w, r, dns.RcodeServerFailure)
			return
		}

		m.ResolveResponseSend(w, r, v.(*model.ResolveResult), do, zk)
	})
	return m
}

func (r *Module) ResolveQuery(ctx context.Context, key, name, qtype string) (*model.ResolveResult, error) {
	result, err := r.Query(ctx, name, qtype)
	if err == nil {
		r.ResolveCacheResult(ctx, key, result)
	}
	return result, err
}

func (r *Module) ResolveCacheResult(ctx context.Context, key string, result *model.ResolveResult) {
	if r.redis == nil || *result.Rcode != model.RcodeNOERROR || result.ExpiredAt == nil {
		return
	}
	ttl := int(time.Until(*result.ExpiredAt).Seconds())
	if *result.Rcode == model.RcodeNOERROR && ttl < 60 {
		ttl = 60
	}
	if data, err := json.Marshal(result); err == nil {
		r.redis.Set(ctx, key, data, time.Duration(ttl)*time.Second)
	}
}

func (r *Module) ResolveResponseSend(w dns.ResponseWriter, msg *dns.Msg, result *model.ResolveResult, do bool, zk *ZoneKey) {
	dnsMsg := buildResponse(msg, result)
	if do && zk != nil {
		if answer := len(dnsMsg.Answer); answer > 0 {
			r.DnssecSign(&dnsMsg.Answer, dnsMsg.Answer)
		} else if dnsMsg.Rcode == dns.RcodeNameError {
			r.DnssecSignNx(dnsMsg, zk)
		} else if dnsMsg.Rcode == dns.RcodeSuccess {
			if len(msg.Question) > 0 {
				q := msg.Question[0]
				r.DnssecSignNodata(dnsMsg, zk, q.Name)
			}
		}
	}
	_ = w.WriteMsg(dnsMsg)
}
