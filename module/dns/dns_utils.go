package dns

import (
	"fmt"
	"time"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/type/model"
)

func newMessage(r *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(r)
	if opt := r.IsEdns0(); opt != nil {
		m.SetEdns0(opt.UDPSize(), false)
	}
	return m
}

func buildResponse(r *dns.Msg, result *model.ResolveResult) *dns.Msg {
	m := newMessage(r)
	m.Authoritative = true

	switch *result.Rcode {
	case model.RcodeNOERROR:
		m.Rcode = dns.RcodeSuccess
		var ttl int
		if result.ExpiredAt != nil {
			ttl = int(time.Until(*result.ExpiredAt).Seconds() - 30)
			if ttl < 0 {
				ttl = 0
			}
		}
		for _, rec := range result.Records {
			rrStr := fmt.Sprintf("%s %d IN %s %s", dns.Fqdn(*rec.Name), ttl, *rec.Type, *rec.Value)
			parsed, err := dns.NewRR(rrStr)
			if err == nil {
				m.Answer = append(m.Answer, parsed)
			}
		}
	case model.RcodeNXDOMAIN:
		m.Rcode = dns.RcodeNameError
	default:
		m.Rcode = dns.RcodeServerFailure
	}

	return m
}

func replyCode(w dns.ResponseWriter, r *dns.Msg, code int) {
	m := newMessage(r)
	m.Rcode = code
	w.WriteMsg(m)
}
