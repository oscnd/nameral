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
	"go.uber.org/fx"
)

func New(lifecycle fx.Lifecycle, polygon polygon.Polygon, cfg *config.Config, server *dns.Server, redis *redis.Client) *Module {
	m := &Module{
		layer:    polygon.Layer("dns", "module"),
		mutex:    new(sync.RWMutex),
		no:       new(atomic.Uint64),
		pending:  new(sync.Map),
		inflight: new(sync.Map),
		stopCh:   make(chan struct{}),
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

			// * if resolution staled, return immediately but re-query in background
			if result.ResolvedAt != nil && time.Since(*result.ResolvedAt) > 30*time.Second {
				m.ResolveResponseSend(w, r, result, do, zk)
				go m.ResolveCacheRefresh(context.Background(), key, name, qtype)
				return
			}

			m.ResolveResponseSend(w, r, result, do, zk)
			return
		}

		// * coalesce concurrent requests for the same key
		if ch, loaded := m.inflight.LoadOrStore(key, make(chan *model.ResolveResult, 1)); loaded {
			// * wait for result
			result := <-ch.(chan *model.ResolveResult)
			if result == nil {
				replyCode(w, r, dns.RcodeServerFailure)
				return
			}
			m.ResolveResponseSend(w, r, result, do, zk)
			return
		}

		// * resolve via module registry
		result, err := m.Query(ctx, name, qtype)
		if err != nil {
			m.inflight.Delete(key)
			replyCode(w, r, dns.RcodeServerFailure)
			return
		}

		m.ResolveCacheResult(ctx, key, result)
		m.ResolveResponseSend(w, r, result, do, zk)

		// * broadcast result to other waiting goroutines and clean up
		if ch, loaded := m.inflight.LoadAndDelete(key); loaded {
			ch.(chan *model.ResolveResult) <- result
		}
	})
	return m
}

func (r *Module) ResolveCacheRefresh(ctx context.Context, key, name, qtype string) {
	newResult, err := r.Query(ctx, name, qtype)
	if err != nil {
		return
	}
	r.ResolveCacheResult(ctx, key, newResult)
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
