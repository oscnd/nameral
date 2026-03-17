package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/client"
	"go.scnd.dev/open/nameral/command/agent/handler"
	recordEndpoint "go.scnd.dev/open/nameral/command/agent/handler/record"
	"go.scnd.dev/open/nameral/command/agent/store"
	"go.scnd.dev/open/nameral/type/model"
	"go.scnd.dev/open/polygon/compat/common"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			fx.Annotate(
				common.Config[Config],
				fx.As(new(common.PolygonConfig)),
				fx.As(new(handler.Config)),
			),
			common.Config[Config],
			common.Polygon,
			provideRecordStore,
			provideFiber,
			recordEndpoint.Handle,
		),
		fx.Invoke(
			handler.Bind,
			invoke,
		),
	).Run()
}

func provideRecordStore(config *Config) *store.Store {
	if config.RecordFile == nil {
		return nil
	}
	s := store.NewStore(*config.RecordFile)
	s.Load()
	return s
}

func provideFiber(lc fx.Lifecycle, config *Config) *fiber.App {
	if config.RecordFile == nil || config.RecordKey == nil || len(config.WebListen) == 0 {
		return nil
	}
	app := fiber.New(fiber.Config{ErrorHandler: common.FiberError})
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			for _, addr := range config.WebListen {
				a := *addr
				go func() {
					_ = app.Listen(a)
				}()
			}
			return nil
		},
		OnStop: func(_ context.Context) error {
			return app.Shutdown()
		},
	})
	return app
}

func invoke(lc fx.Lifecycle, config *Config, s *store.Store) error {
	var tlsConfig *tls.Config
	if config.CertificateFile != nil {
		pem, err := os.ReadFile(*config.CertificateFile)
		if err != nil {
			return err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		tlsConfig = &tls.Config{RootCAs: pool}
	}

	namera, err := client.New(&client.Config{
		Address: config.Address,
		Secret:  config.Secret,
		Tls:     tlsConfig,
	})
	if err != nil {
		return err
	}

	dnsClient := &dns.Client{}

	if s != nil {
		go s.Tick()
	}

	for _, zone := range config.Zones {
		z := *zone
		namera.Handle(z, func(q *model.HandleQuery) (*model.HandleResponse, error) {
			// Build FQDN
			var fqdn string
			if *q.Zone == "." {
				fqdn = dns.Fqdn(*q.Subdomain)
			} else if q.Subdomain != nil && *q.Subdomain != "" {
				fqdn = dns.Fqdn(*q.Subdomain + "." + *q.Zone)
			} else {
				fqdn = dns.Fqdn(*q.Zone)
			}

			qtype, ok := dns.StringToType[*q.Type]
			if !ok {
				rcode := "SERVFAIL"
				return &model.HandleResponse{Rcode: &rcode}, nil
			}

			// Check record store first
			if s != nil {
				s.Mu.RLock()
				records, found := s.Records[fqdn]
				s.Mu.RUnlock()

				if found {
					var matched []*model.Record
					for _, r := range records {
						if *r.Type == *q.Type || (*q.Type != "CNAME" && *r.Type == "CNAME") {
							name := strings.TrimSuffix(fqdn, ".")
							typ := *r.Type
							vals := make([]string, len(r.Values))
							for i, v := range r.Values {
								vals[i] = *v
							}
							val := strings.Join(vals, " ")
							matched = append(matched, &model.Record{
								Name:  &name,
								Type:  &typ,
								Value: &val,
							})
						}
					}
					if len(matched) > 0 {
						rcode := "NOERROR"
						ttl := 30
						return &model.HandleResponse{
							Rcode:   &rcode,
							Ttl:     &ttl,
							Records: matched,
						}, nil
					}
				}
			}

			// Forward to upstream
			if config.Upstream != nil {
				upstream := *config.Upstream
				if !strings.Contains(upstream, ":") {
					upstream += ":53"
				}

				m := new(dns.Msg)
				m.SetQuestion(fqdn, qtype)
				m.RecursionDesired = true

				r, _, err := dnsClient.Exchange(m, upstream)
				if err != nil || r == nil {
					rcode := "SERVFAIL"
					return &model.HandleResponse{Rcode: &rcode}, nil
				}

				if r.Rcode == dns.RcodeNameError {
					rcode := "NXDOMAIN"
					return &model.HandleResponse{Rcode: &rcode}, nil
				}

				if r.Rcode != dns.RcodeSuccess || len(r.Answer) == 0 {
					rcode := "SERVFAIL"
					return &model.HandleResponse{Rcode: &rcode}, nil
				}

				rcode := "NOERROR"
				resp := &model.HandleResponse{Rcode: &rcode}

				for _, rr := range r.Answer {
					hdr := rr.Header()
					rrName := strings.TrimSuffix(hdr.Name, ".")
					rrType := dns.TypeToString[hdr.Rrtype]
					ttl := int(hdr.Ttl)

					full := rr.String()
					parts := strings.Fields(full)
					var value string
					if len(parts) >= 5 {
						value = strings.Join(parts[4:], " ")
					}

					resp.Ttl = &ttl
					resp.Records = append(resp.Records, &model.Record{
						Name:  &rrName,
						Type:  &rrType,
						Value: &value,
					})
				}

				return resp, nil
			}

			// Neither store hit nor upstream
			rcode := "NXDOMAIN"
			return &model.HandleResponse{Rcode: &rcode}, nil
		})
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			if s != nil {
				s.Stop()
			}
			return namera.Close()
		},
	})

	return nil
}
