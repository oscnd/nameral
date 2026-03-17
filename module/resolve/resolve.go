package resolve

import (
	"strings"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/module/store"
	"go.scnd.dev/open/nameral/type/model"
)

type Resolve struct {
	Store     *store.Store
	DnsClient *dns.Client
	Upstream  *string
}

func (r *Resolve) Handle(q *model.HandleQuery) (*model.HandleResponse, error) {
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
	if r.Store != nil {
		r.Store.Mu.RLock()
		matched, nameFound := r.resolveStore(r.Store.Records, fqdn, *q.Type)
		r.Store.Mu.RUnlock()

		if nameFound {
			if len(matched) > 0 {
				rcode := "NOERROR"
				ttl := 30
				return &model.HandleResponse{Rcode: &rcode, Ttl: &ttl, Records: matched}, nil
			}
			rcode := "NXDOMAIN"
			return &model.HandleResponse{Rcode: &rcode}, nil
		}
	}

	// Forward to upstream
	if r.Upstream != nil {
		upstream := *r.Upstream
		if !strings.Contains(upstream, ":") {
			upstream += ":53"
		}

		m := new(dns.Msg)
		m.SetQuestion(fqdn, qtype)
		m.RecursionDesired = true

		msg, _, err := r.DnsClient.Exchange(m, upstream)
		if err != nil || msg == nil {
			rcode := "SERVFAIL"
			return &model.HandleResponse{Rcode: &rcode}, nil
		}

		if msg.Rcode == dns.RcodeNameError {
			rcode := "NXDOMAIN"
			return &model.HandleResponse{Rcode: &rcode}, nil
		}

		if msg.Rcode != dns.RcodeSuccess || len(msg.Answer) == 0 {
			rcode := "SERVFAIL"
			return &model.HandleResponse{Rcode: &rcode}, nil
		}

		rcode := "NOERROR"
		resp := &model.HandleResponse{Rcode: &rcode}

		for _, rr := range msg.Answer {
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

	rcode := "NXDOMAIN"
	return &model.HandleResponse{Rcode: &rcode}, nil
}
