package dns

import (
	"crypto"

	"github.com/miekg/dns"
)

type ZoneKey struct {
	dnsKey     *dns.DNSKEY
	privateKey crypto.PrivateKey
}

func (r *ZoneKey) Ds() *dns.DS {
	return r.dnsKey.ToDS(dns.SHA256)
}
