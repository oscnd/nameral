package resolve

import (
	"strings"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/type/model"
)

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
