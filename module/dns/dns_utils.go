package dns

import (
	"fmt"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/generate/proto"
)

func writeResponse(w dns.ResponseWriter, r *dns.Msg, result *proto.ResolveResult) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	switch result.Rcode {
	case "NOERROR":
		m.Rcode = dns.RcodeSuccess
		for _, rr := range result.Rrs {
			rrStr := fmt.Sprintf("%s %d IN %s %s", dns.Fqdn(rr.Name), result.Ttl, rr.Type, rr.Value)
			parsed, err := dns.NewRR(rrStr)
			if err == nil {
				m.Answer = append(m.Answer, parsed)
			}
		}
	case "NXDOMAIN":
		m.Rcode = dns.RcodeNameError
	default:
		m.Rcode = dns.RcodeServerFailure
	}

	w.WriteMsg(m)
}

func replyCode(w dns.ResponseWriter, r *dns.Msg, code int) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Rcode = code
	w.WriteMsg(m)
}
