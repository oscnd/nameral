package model

type HandleQuery struct {
	Type      *string
	Zone      *string
	Subdomain *string
}

type HandleResponse struct {
	Rcode   *string
	Ttl     *int
	Records []*Record
}

type Record struct {
	Name  *string
	Type  *string
	Value *string
}
