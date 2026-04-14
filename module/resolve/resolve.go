package resolve

import (
	"strings"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/module/store"
	"go.scnd.dev/open/nameral/type/model"
)

type Resolve struct {
	Store           *store.Store
	DnsClient       *dns.Client
	Upstream        *string
	UpstreamRewrite []*string
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
		return &model.HandleResponse{Rcode: &model.RcodeSERVFAIL}, nil
	}

	// Check record store first
	if r.Store != nil {
		r.Store.Mu.RLock()
		matched, nameFound := r.resolveStore(r.Store.Records, fqdn, *q.Type)
		r.Store.Mu.RUnlock()

		if nameFound {
			if len(matched) > 0 {
				ttl := 30
				return &model.HandleResponse{Rcode: &model.RcodeNOERROR, Ttl: &ttl, Records: matched}, nil
			}
			return &model.HandleResponse{Rcode: &model.RcodeNOERROR}, nil
		}
	}

	// Forward to upstream
	if r.Upstream != nil {
		upstream := *r.Upstream
		if !strings.Contains(upstream, ":") {
			upstream += ":53"
		}

		// Apply UpstreamRewrite if configured
		upstreamFqdn := fqdn
		for i := 0; i+1 < len(r.UpstreamRewrite); i += 2 {
			from := dns.Fqdn(*r.UpstreamRewrite[i])
			to := dns.Fqdn(*r.UpstreamRewrite[i+1])
			if strings.HasSuffix(upstreamFqdn, from) {
				upstreamFqdn = strings.TrimSuffix(upstreamFqdn, from) + to
				break
			}
		}

		m := new(dns.Msg)
		m.SetQuestion(upstreamFqdn, qtype)
		m.RecursionDesired = true

		msg, _, err := r.DnsClient.Exchange(m, upstream)
		if err != nil || msg == nil {
			return &model.HandleResponse{Rcode: &model.RcodeSERVFAIL}, nil
		}

		if msg.Rcode == dns.RcodeNameError {
			return &model.HandleResponse{Rcode: &model.RcodeNXDOMAIN}, nil
		}

		if msg.Rcode != dns.RcodeSuccess {
			return &model.HandleResponse{Rcode: &model.RcodeSERVFAIL}, nil
		}

		if len(msg.Answer) == 0 {
			return &model.HandleResponse{Rcode: &model.RcodeNOERROR}, nil
		}

		resp := &model.HandleResponse{Rcode: &model.RcodeNOERROR}

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

	return &model.HandleResponse{Rcode: &model.RcodeNXDOMAIN}, nil
}
