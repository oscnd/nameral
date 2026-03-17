package resolve

import (
	"strings"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/module/store"
	"go.scnd.dev/open/nameral/type/model"
	"go.scnd.dev/open/nameral/type/payload"
	"go.scnd.dev/open/nameral/util"
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

func (r *Resolve) resolveStore(records map[string][]*payload.Record, fqdn string, qtype string) (matched []*model.Record, nameFound bool) {
	entries := records[fqdn]
	if len(entries) == 0 {
		return nil, false
	}
	nameFound = true

	name := strings.TrimSuffix(fqdn, ".")

	switch qtype {
	case "A":
		// If CNAME exists, redirect entirely to CNAME resolution
		for _, rec := range entries {
			if *rec.Type == "CNAME" {
				return r.resolveStore(records, fqdn, "CNAME")
			}
		}
		for _, rec := range entries {
			if *rec.Type != "A" {
				continue
			}
			typ := "A"
			val := util.JoinValues(rec.Values)
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	case "CNAME":
		// If A records exist, return them directly
		for _, rec := range entries {
			if *rec.Type == "A" {
				typ := "A"
				val := util.JoinValues(rec.Values)
				matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
			}
		}
		if len(matched) > 0 {
			return
		}
		// Return CNAME RR(s) and resolve the target (store first, then upstream)
		for _, rec := range entries {
			if *rec.Type != "CNAME" {
				continue
			}
			typ := "CNAME"
			target := util.JoinValues(rec.Values)
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &target})

			targetFqdn := dns.Fqdn(target)
			if targetFqdn == fqdn {
				continue
			}
			more, targetFound := r.resolveStore(records, targetFqdn, "CNAME")
			if targetFound {
				matched = append(matched, more...)
			} else if r.Upstream != nil {
				matched = append(matched, r.resolveUpstream(targetFqdn, "A")...)
			}
		}

	case "MX":
		for _, rec := range entries {
			if *rec.Type != "MX" {
				continue
			}
			typ := "MX"
			val := util.JoinValues(rec.Values)
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	case "TXT":
		for _, rec := range entries {
			if *rec.Type != "TXT" {
				continue
			}
			typ := "TXT"
			val := util.JoinValues(rec.Values)
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	default:
		for _, rec := range entries {
			if *rec.Type != qtype {
				continue
			}
			typ := qtype
			val := util.JoinValues(rec.Values)
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}
	}

	return
}

func (r *Resolve) resolveUpstream(fqdn string, qtype string) []*model.Record {
	if r.Upstream == nil || r.DnsClient == nil {
		return nil
	}

	upstream := *r.Upstream
	if !strings.Contains(upstream, ":") {
		upstream += ":53"
	}

	qt, ok := dns.StringToType[qtype]
	if !ok {
		return nil
	}

	m := new(dns.Msg)
	m.SetQuestion(fqdn, qt)
	m.RecursionDesired = true

	msg, _, err := r.DnsClient.Exchange(m, upstream)
	if err != nil || msg == nil || msg.Rcode != dns.RcodeSuccess {
		return nil
	}

	var result []*model.Record
	for _, rr := range msg.Answer {
		hdr := rr.Header()
		rrName := strings.TrimSuffix(hdr.Name, ".")
		rrType := dns.TypeToString[hdr.Rrtype]

		full := rr.String()
		parts := strings.Fields(full)
		var value string
		if len(parts) >= 5 {
			value = strings.Join(parts[4:], " ")
		}

		result = append(result, &model.Record{
			Name:  &rrName,
			Type:  &rrType,
			Value: &value,
		})
	}

	return result
}
