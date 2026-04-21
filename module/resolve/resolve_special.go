package resolve

import (
	"strings"

	"go.scnd.dev/open/nameral/type/model"
	"go.scnd.dev/open/polygon/utility/value"
)

func (r *Resolve) BuildSoa(fqdn string) *model.HandleResponse {
	if r.DefaultSoa == nil {
		return nil
	}
	ttl := 300
	zoneName := strings.TrimSuffix(fqdn, ".")
	return &model.HandleResponse{
		Rcode: &model.RcodeNOERROR,
		Ttl:   &ttl,
		Records: []*model.Record{
			{
				Name:  &zoneName,
				Type:  value.Ptr("SOA"),
				Value: r.DefaultSoa,
			},
		},
	}
}
