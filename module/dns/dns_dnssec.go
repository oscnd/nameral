package dns

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/miekg/dns"
)

func loadZoneKey(dnssecPath, zone string) (*ZoneKey, error) {
	fqdn := dns.Fqdn(zone)
	base := strings.TrimSuffix(fqdn, ".")
	keyFile := filepath.Join(dnssecPath, base+".key")
	privFile := filepath.Join(dnssecPath, base+".private")

	// Try to load existing keys
	if keyData, err := os.ReadFile(keyFile); err == nil {
		if rr, err := dns.NewRR(strings.TrimSpace(string(keyData))); err == nil {
			if dnskey, ok := rr.(*dns.DNSKEY); ok {
				if f, err := os.Open(privFile); err == nil {
					defer f.Close()
					if pk, err := dnskey.ReadPrivateKey(f, privFile); err == nil {
						return &ZoneKey{dnsKey: dnskey, privateKey: pk}, nil
					}
				}
			}
		}
	}

	// Generate new key
	if err := os.MkdirAll(dnssecPath, 0755); err != nil {
		return nil, fmt.Errorf("create dnssec dir: %w", err)
	}

	dnskey := &dns.DNSKEY{
		Hdr: dns.RR_Header{
			Name:   fqdn,
			Rrtype: dns.TypeDNSKEY,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Flags:     257,
		Protocol:  3,
		Algorithm: dns.ED25519,
	}

	pk, err := dnskey.Generate(256)
	if err != nil {
		return nil, fmt.Errorf("generate key for %s: %w", zone, err)
	}

	if err := os.WriteFile(keyFile, []byte(dnskey.String()+"\n"), 0644); err != nil {
		return nil, fmt.Errorf("save public key: %w", err)
	}

	privStr := dnskey.PrivateKeyString(pk)
	if err := os.WriteFile(privFile, []byte(privStr), 0600); err != nil {
		return nil, fmt.Errorf("save private key: %w", err)
	}

	return &ZoneKey{dnsKey: dnskey, privateKey: pk}, nil
}
