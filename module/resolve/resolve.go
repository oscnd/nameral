package resolve

import (
	"regexp"
	"strings"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/module/store"
	"go.scnd.dev/open/nameral/type/model"
)

type Resolve struct {
	Store           *store.Store
	DnsClient       *dns.Client
	UpstreamAddress *string
	UpstreamFrom    *string
	UpstreamTo      *string
	DefaultSoa      *string
}

func (r *Resolve) Handle(q *model.HandleQuery) (*model.HandleResponse, error) {
	// * build fqdn
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

	// * check record store first
	if r.Store != nil {
		r.Store.Mu.RLock()
		matched, nameFound := r.resolveStore(r.Store.Records, fqdn, *q.Type)
		r.Store.Mu.RUnlock()

		if nameFound {
			if len(matched) > 0 {
				ttl := 60
				return &model.HandleResponse{
					Rcode:   &model.RcodeNOERROR,
					Ttl:     &ttl,
					Records: matched,
				}, nil
			}
			if resp := r.BuildSoa(fqdn); resp != nil {
				return resp, nil
			}
			return &model.HandleResponse{
				Rcode:   &model.RcodeNOERROR,
				Ttl:     nil,
				Records: nil,
			}, nil
		}
	}

	// * forward to upstream
	if r.UpstreamAddress != nil && r.UpstreamFrom != nil && r.UpstreamTo != nil {
		if !strings.Contains(*r.UpstreamAddress, ":") {
			*r.UpstreamAddress += ":53"
		}

		// * apply rewrite if configured
		upstreamFqdn := fqdn
		re := regexp.MustCompile(*r.UpstreamFrom)
		if re.MatchString(upstreamFqdn) {
			upstreamFqdn = re.ReplaceAllString(upstreamFqdn, *r.UpstreamTo)
		} else {
			// * if not match, ignore upstream
			return r.NxDomainResponse(*q.Zone), nil
		}

		m := new(dns.Msg)
		m.SetQuestion(upstreamFqdn, qtype)

		msg, _, err := r.DnsClient.Exchange(m, *r.UpstreamAddress)
		if err != nil {
			return &model.HandleResponse{
				Rcode:   &model.RcodeSERVFAIL,
				Ttl:     nil,
				Records: nil,
			}, nil
		}

		if msg.Rcode == dns.RcodeNameError {
			return r.NxDomainResponse(*q.Zone), nil
		}

		if msg.Rcode != dns.RcodeSuccess {
			if msg.Rcode == dns.RcodeServerFailure {
				if qtype == dns.TypeA {
					return &model.HandleResponse{
						Rcode:   &model.RcodeNXDOMAIN,
						Ttl:     nil,
						Records: nil,
					}, nil
				}

				// * retry with a query
				mA := new(dns.Msg)
				mA.SetQuestion(upstreamFqdn, dns.TypeA)
				msgA, _, errA := r.DnsClient.Exchange(mA, *r.UpstreamAddress)
				if errA != nil || len(msgA.Answer) == 0 {
					return r.NxDomainResponse(*q.Zone), nil
				}

				// * if a query succeeded, skip to noerror instead
				if msgA.Rcode == dns.RcodeSuccess && len(msgA.Answer) > 0 {
					goto answer
				}

				return &model.HandleResponse{
					Rcode:   &model.RcodeSERVFAIL,
					Ttl:     nil,
					Records: nil,
				}, nil
			}

			return &model.HandleResponse{
				Rcode:   &model.RcodeSERVFAIL,
				Ttl:     nil,
				Records: nil,
			}, nil
		}

	answer:
		if len(msg.Answer) == 0 {
			if qtype == dns.TypeSOA {
				if resp := r.BuildSoa(fqdn); resp != nil {
					return resp, nil
				}
			}
			return &model.HandleResponse{
				Rcode:   &model.RcodeNOERROR,
				Ttl:     nil,
				Records: nil,
			}, nil
		}

		resp := &model.HandleResponse{
			Rcode:   &model.RcodeNOERROR,
			Ttl:     nil,
			Records: nil,
		}

		for _, rr := range msg.Answer {
			hdr := rr.Header()
			rrName := strings.TrimSuffix(hdr.Name, ".")
			rrType := dns.TypeToString[hdr.Rrtype]
			ttl := int(hdr.Ttl)

			full := rr.String()
			parts := strings.Fields(full)
			var rrValue string
			if len(parts) >= 5 {
				rrValue = strings.Join(parts[4:], " ")
			}

			resp.Ttl = &ttl
			resp.Records = append(resp.Records, &model.Record{
				Name:  &rrName,
				Type:  &rrType,
				Value: &rrValue,
			})
		}

		return resp, nil
	}

	return r.NxDomainResponse(*q.Zone), nil
}
