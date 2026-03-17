package main

import (
	"context"
	"strings"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/client"
	"go.scnd.dev/open/nameral/type/model"
	"go.scnd.dev/open/polygon/compat/common"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			common.Config[Config],
		),
		fx.Invoke(
			invoke,
		),
	).Run()
}

func invoke(lc fx.Lifecycle, config *Config) error {
	namera, err := client.New(&client.Config{
		Address: config.Address,
		Secret:  config.Secret,
	})
	if err != nil {
		return err
	}

	dnsClient := &dns.Client{}

	for _, zone := range config.Zones {
		z := *zone
		namera.Handle(z, func(q *model.HandleQuery) (*model.HandleResponse, error) {
			// Build the full DNS name
			name := *q.Zone
			if q.Subdomain != nil && *q.Subdomain != "" {
				name = *q.Subdomain + "." + *q.Zone
			}

			qtype, ok := dns.StringToType[*q.Type]
			if !ok {
				rcode := "SERVFAIL"
				return &model.HandleResponse{Rcode: &rcode}, nil
			}

			m := new(dns.Msg)
			m.SetQuestion(dns.Fqdn(name), qtype)
			m.RecursionDesired = true

			upstream := *config.Upstream
			if !strings.Contains(upstream, ":") {
				upstream += ":53"
			}

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

				// Extract value: everything after the header in the string representation
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
		})
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return namera.Close()
		},
	})

	return nil
}
