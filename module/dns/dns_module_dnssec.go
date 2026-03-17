package dns

import (
	"crypto"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func (r *Module) dnssecMatchZone(name string) *ZoneKey {
	var best *ZoneKey
	bestLen := 0
	for zone, z := range r.registry {
		if z.dnssecZoneKey == nil {
			continue
		}
		if (name == zone || strings.HasSuffix(name, "."+zone)) && len(zone) > bestLen {
			best = z.dnssecZoneKey
			bestLen = len(zone)
		}
	}
	return best
}

func (r *Module) dnssecSign(msg *dns.Msg, rrset []dns.RR, zk *ZoneKey) {
	if zk == nil || len(rrset) == 0 {
		return
	}

	type rrKey struct {
		name   string
		rrtype uint16
	}
	groups := make(map[rrKey][]dns.RR)
	var order []rrKey
	for _, rr := range rrset {
		k := rrKey{rr.Header().Name, rr.Header().Rrtype}
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		groups[k] = append(groups[k], rr)
	}

	now := time.Now().UTC()
	for _, k := range order {
		group := groups[k]
		rrsig := &dns.RRSIG{
			Hdr: dns.RR_Header{
				Name:   group[0].Header().Name,
				Rrtype: dns.TypeRRSIG,
				Class:  dns.ClassINET,
				Ttl:    group[0].Header().Ttl,
			},
			TypeCovered: group[0].Header().Rrtype,
			Algorithm:   zk.dnsKey.Algorithm,
			Labels:      uint8(dns.CountLabel(group[0].Header().Name)),
			OrigTtl:     group[0].Header().Ttl,
			Expiration:  uint32(now.Add(24 * time.Hour).Unix()),
			Inception:   uint32(now.Add(-1 * time.Minute).Unix()),
			KeyTag:      zk.dnsKey.KeyTag(),
			SignerName:  zk.dnsKey.Hdr.Name,
		}
		if err := rrsig.Sign(zk.privateKey.(crypto.Signer), group); err == nil {
			msg.Answer = append(msg.Answer, rrsig)
		}
	}
}
