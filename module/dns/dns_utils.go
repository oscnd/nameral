package dns

import (
	"fmt"

	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/generate/proto"
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

func buildResponse(r *dns.Msg, result *proto.ResolveResult) *dns.Msg {
	m := newMessage(r)
	m.Authoritative = true

	switch model.Rcode(result.Rcode) {
	case model.RcodeNOERROR:
		m.Rcode = dns.RcodeSuccess
		for _, rr := range result.Rrs {
			rrStr := fmt.Sprintf("%s %d IN %s %s", dns.Fqdn(rr.Name), result.Ttl, rr.Type, rr.Value)
			parsed, err := dns.NewRR(rrStr)
			if err == nil {
				m.Answer = append(m.Answer, parsed)
			}
		}
	case (model.RcodeNXDOMAIN):
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
