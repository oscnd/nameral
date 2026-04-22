package resolve

import (
	"strings"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/type/model"
	"go.scnd.dev/open/nameral/type/payload"
)

func (r *Resolve) resolveStore(records map[string]*payload.Record, fqdn string, qtype string, zone string) (matched []*model.Record, nameFound bool) {
	var entries []*payload.Record
	for _, rec := range records {
		if *rec.Name == fqdn {
			entries = append(entries, rec)
		}
	}
	if len(entries) == 0 {
		return nil, false
	}
	nameFound = true

	name := strings.TrimSuffix(fqdn, ".")

	switch qtype {
	case "A":
		for _, rec := range entries {
			if *rec.Type == "CNAME" {
				return r.resolveStore(records, fqdn, "CNAME", zone)
			}
		}
		for _, rec := range entries {
			if *rec.Type != "A" {
				continue
			}
			typ := "A"
			val := *rec.Value
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	case "CNAME":
		for _, rec := range entries {
			if *rec.Type == "A" {
				typ := "A"
				val := *rec.Value
				matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
			}
		}
		if len(matched) > 0 {
			return
		}
		for _, rec := range entries {
			if *rec.Type != "CNAME" {
				continue
			}
			typ := "CNAME"
			target := *rec.Value
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &target})

			targetFqdn := dns.Fqdn(target)
			if targetFqdn == fqdn {
				continue
			}

			zoneFqdn := dns.Fqdn(zone)
			inZone := targetFqdn == zoneFqdn || strings.HasSuffix(targetFqdn, "."+zoneFqdn)
			if !inZone {
				continue
			}

			more, targetFound := r.resolveStore(records, targetFqdn, "CNAME", zone)
			if targetFound {
				matched = append(matched, more...)
			}
		}

	case "MX":
		for _, rec := range entries {
			if *rec.Type != "MX" {
				continue
			}
			typ := "MX"
			val := *rec.Value
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	case "NS":
		for _, rec := range entries {
			if *rec.Type != "NS" {
				continue
			}
			typ := "NS"
			val := *rec.Value
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	case "TXT":
		for _, rec := range entries {
			if *rec.Type != "TXT" {
				continue
			}
			typ := "TXT"
			val := *rec.Value
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	case "SOA":
		for _, rec := range entries {
			if *rec.Type != "SOA" {
				continue
			}
			typ := "SOA"
			val := *rec.Value
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}

	default:
		for _, rec := range entries {
			if *rec.Type != qtype {
				continue
			}
			typ := qtype
			val := *rec.Value
			matched = append(matched, &model.Record{Name: &name, Type: &typ, Value: &val})
		}
	}

	return
}
