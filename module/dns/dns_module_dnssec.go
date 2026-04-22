package dns

import (
	"crypto"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func (r *Module) DnssecSign(target *[]dns.RR, rrset []dns.RR, zk *ZoneKey) {
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
			Expiration:  uint32(now.Add(1 * time.Hour).Unix()),
			Inception:   uint32(now.Add(-1 * time.Minute).Unix()),
			KeyTag:      zk.dnsKey.KeyTag(),
			SignerName:  zk.dnsKey.Hdr.Name,
		}
		if err := rrsig.Sign(zk.privateKey.(crypto.Signer), group); err == nil {
			*target = append(*target, rrsig)
		}
	}
}

func (r *Module) DnssecSignNx(dnsMsg *dns.Msg, zk *ZoneKey) {
	zoneName := dns.Fqdn(zk.dnsKey.Hdr.Name)
	nsec := &dns.NSEC{
		Hdr: dns.RR_Header{
			Name:   zoneName,
			Rrtype: dns.TypeNSEC,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		NextDomain: zoneName,
		TypeBitMap: []uint16{dns.TypeSOA, dns.TypeRRSIG, dns.TypeNSEC, dns.TypeDNSKEY},
	}

	dnsMsg.Ns = append(dnsMsg.Ns, nsec)

	now := time.Now().UTC()
	rrsig := &dns.RRSIG{
		Hdr: dns.RR_Header{
			Name:   zoneName,
			Rrtype: dns.TypeRRSIG,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		TypeCovered: dns.TypeNSEC,
		Algorithm:   zk.dnsKey.Algorithm,
		Labels:      uint8(dns.CountLabel(zoneName)),
		OrigTtl:     300,
		Expiration:  uint32(now.Add(1 * time.Hour).Unix()),
		Inception:   uint32(now.Add(-1 * time.Minute).Unix()),
		KeyTag:      zk.dnsKey.KeyTag(),
		SignerName:  zoneName,
	}

	if err := rrsig.Sign(zk.privateKey.(crypto.Signer), []dns.RR{dns.Copy(nsec)}); err == nil {
		dnsMsg.Ns = append(dnsMsg.Ns, rrsig)
	}
}

func (r *Module) DnssecMatchZone(name string) *ZoneKey {
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
