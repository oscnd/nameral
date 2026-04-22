package dns

import (
	"crypto"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func (r *Module) DnssecSign(target *[]dns.RR, rrset []dns.RR) {
	if len(rrset) == 0 {
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
		name := strings.TrimSuffix(strings.ToLower(k.name), ".")
		zk := r.DnssecMatchZone(name)
		if zk == nil {
			continue
		}
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

func (r *Module) DnssecSignNsec(dnsMsg *dns.Msg, nsec *dns.NSEC, name, signerName string, algorithm uint8, keyTag uint16, privateKey crypto.PrivateKey) {
	now := time.Now().UTC()
	rrsig := &dns.RRSIG{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeRRSIG,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		TypeCovered: dns.TypeNSEC,
		Algorithm:   algorithm,
		Labels:      uint8(dns.CountLabel(name)),
		OrigTtl:     300,
		Expiration:  uint32(now.Add(1 * time.Hour).Unix()),
		Inception:   uint32(now.Add(-1 * time.Minute).Unix()),
		KeyTag:      keyTag,
		SignerName:  signerName,
	}

	if err := rrsig.Sign(privateKey.(crypto.Signer), []dns.RR{dns.Copy(nsec)}); err == nil {
		dnsMsg.Ns = append(dnsMsg.Ns, rrsig)
	}
}

func (r *Module) DnssecSignAnswer(dnsMsg *dns.Msg, zk *ZoneKey, qname string, answerTypes []uint16) {
	zoneName := dns.Fqdn(zk.dnsKey.Hdr.Name)
	qnameFqdn := dns.Fqdn(qname)

	typeBitMap := append(answerTypes, dns.TypeRRSIG, dns.TypeNSEC, dns.TypeDNSKEY)
	nsec := &dns.NSEC{
		Hdr: dns.RR_Header{
			Name:   qnameFqdn,
			Rrtype: dns.TypeNSEC,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		NextDomain: zoneName,
		TypeBitMap: typeBitMap,
	}

	dnsMsg.Ns = append(dnsMsg.Ns, nsec)
	r.DnssecSignNsec(dnsMsg, nsec, qnameFqdn, zoneName, zk.dnsKey.Algorithm, zk.dnsKey.KeyTag(), zk.privateKey)
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
	r.DnssecSignNsec(dnsMsg, nsec, zoneName, zoneName, zk.dnsKey.Algorithm, zk.dnsKey.KeyTag(), zk.privateKey)

	// * also sign soa records in authority section
	var soaRecords []dns.RR
	for _, rr := range dnsMsg.Ns {
		if rr.Header().Rrtype == dns.TypeSOA {
			soaRecords = append(soaRecords, rr)
		}
	}
	if len(soaRecords) > 0 {
		r.DnssecSign(&dnsMsg.Ns, soaRecords)
	}
}

func (r *Module) DnssecSignNodata(dnsMsg *dns.Msg, zk *ZoneKey, qname string) {
	zoneName := dns.Fqdn(zk.dnsKey.Hdr.Name)
	qnameFqdn := dns.Fqdn(qname)

	typeBitMap := []uint16{dns.TypeSOA, dns.TypeNS, dns.TypeRRSIG, dns.TypeNSEC, dns.TypeDNSKEY}
	nsec := &dns.NSEC{
		Hdr: dns.RR_Header{
			Name:   qnameFqdn,
			Rrtype: dns.TypeNSEC,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		NextDomain: zoneName,
		TypeBitMap: typeBitMap,
	}

	dnsMsg.Ns = append(dnsMsg.Ns, nsec)
	r.DnssecSignNsec(dnsMsg, nsec, qnameFqdn, zoneName, zk.dnsKey.Algorithm, zk.dnsKey.KeyTag(), zk.privateKey)

	// * also sign soa records in authority section
	var soaRecords []dns.RR
	for _, rr := range dnsMsg.Ns {
		if rr.Header().Rrtype == dns.TypeSOA {
			soaRecords = append(soaRecords, rr)
		}
	}
	if len(soaRecords) > 0 {
		r.DnssecSign(&dnsMsg.Ns, soaRecords)
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
