package model

import "time"

type Rcode string

var (
	RcodeNOERROR  Rcode = "NOERROR"
	RcodeSERVFAIL Rcode = "SERVFAIL"
	RcodeNXDOMAIN Rcode = "NXDOMAIN"
)

type HandleQuery struct {
	Type      *string `json:"type"`
	Zone      *string `json:"zone"`
	Subdomain *string `json:"subdomain"`
}

type HandleResponse struct {
	Rcode   *Rcode    `json:"rcode"`
	Ttl     *int      `json:"ttl"`
	Records []*Record `json:"records"`
}

type Record struct {
	Name  *string `json:"name"`
	Type  *string `json:"type"`
	Value *string `json:"value"`
}

type ResolveResult struct {
	No         *uint64    `json:"no"`
	Rcode      *Rcode     `json:"rcode"`
	ResolvedAt *time.Time `json:"resolvedAt"`
	ExpiredAt  *time.Time `json:"expiredAt"`
	Records    []*Record  `json:"records"`
}
