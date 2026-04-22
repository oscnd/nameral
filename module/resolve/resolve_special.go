package resolve

import (
	"strings"

	"go.scnd.dev/open/nameral/type/model"
	"go.scnd.dev/open/polygon/utility/value"
)

func (r *Resolve) BuildSoa(zone *string) *model.HandleResponse {
	if r.DefaultSoa == nil {
		return nil
	}
	ttl := 300
	return &model.HandleResponse{
		Rcode: &model.RcodeNOERROR,
		Ttl:   &ttl,
		Records: []*model.Record{
			{
				Name:  zone,
				Type:  value.Ptr("SOA"),
				Value: r.DefaultSoa,
			},
		},
	}
}

func (r *Resolve) NxDomainResponse(zone string) *model.HandleResponse {
	resp := &model.HandleResponse{Rcode: &model.RcodeNXDOMAIN}
	if r.DefaultSoa == nil {
		return resp
	}
	ttl := 300
	zoneName := strings.TrimSuffix(zone, ".")
	resp.Ttl = &ttl
	resp.Records = []*model.Record{
		{
			Name:  &zoneName,
			Type:  value.Ptr("SOA"),
			Value: r.DefaultSoa,
		},
	}
	return resp
}
