package resolve

import (
	"strings"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/type/model"
	"go.scnd.dev/open/nameral/type/payload"
	"go.scnd.dev/open/nameral/util"
)

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

	case "NS":
		for _, rec := range entries {
			if *rec.Type != "NS" {
				continue
			}
			typ := "NS"
			val := util.JoinValues(rec.Values)
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	case "SOA":
		for _, rec := range entries {
			if *rec.Type != "SOA" {
				continue
			}
			typ := "SOA"
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
