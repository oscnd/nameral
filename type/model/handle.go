package model

type Rcode string

var (
	RcodeNOERROR  Rcode = "NOERROR"
	RcodeSERVFAIL Rcode = "SERVFAIL"
	RcodeNXDOMAIN Rcode = "NXDOMAIN"
)

type HandleQuery struct {
	Type      *string
	Zone      *string
	Subdomain *string
}

type HandleResponse struct {
	Rcode   *Rcode
	Ttl     *int
	Records []*Record
}

type Record struct {
	Name  *string
	Type  *string
	Value *string
}
