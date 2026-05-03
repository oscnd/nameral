package dns

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/nameral/type/model"
)

func (r *Module) Query(ctx context.Context, name string, qtype string) (*model.ResolveResult, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// * collect matching zones
	r.mutex.RLock()
	var matchingZones []string
	for zone := range r.registry {
		if zone == "." || name == zone || strings.HasSuffix(name, "."+zone) {
			matchingZones = append(matchingZones, zone)
		}
	}
	r.mutex.RUnlock()

	if len(matchingZones) == 0 {
		return &model.ResolveResult{Rcode: &model.RcodeSERVFAIL}, nil
	}

	// * sort by zone length descending
	sort.Slice(matchingZones, func(i, j int) bool {
		return len(matchingZones[i]) > len(matchingZones[j])
	})

	var nxResult *model.ResolveResult
	resolved := make(map[*ClientStream]struct{})

	for _, zone := range matchingZones {
		var clients []*ClientStream
		r.mutex.RLock()
		if z := r.registry[zone]; z != nil {
			clients = z.clients
		}
		r.mutex.RUnlock()

		for _, client := range clients {
			if _, ok := resolved[client]; ok {
				continue
			}

			res, err := r.QueryZone(ctx, client, qtype, zone, name)
			if err != nil {
				return nil, err
			}
			if res == nil {
				continue
			}

			if res.Rcode == string(model.RcodeNOERROR) {
				return MapperResolveResult(res), nil
			}
			if res.Rcode == string(model.RcodeNXDOMAIN) && nxResult == nil {
				nxResult = MapperResolveResult(res)
			}
			resolved[client] = struct{}{}
		}
	}

	if nxResult != nil {
		return nxResult, nil
	}

	return &model.ResolveResult{Rcode: &model.RcodeSERVFAIL}, nil
}

func (r *Module) QueryZone(ctx context.Context, client *ClientStream, qtype, zone, name string) (*proto.ResolveResult, error) {
	// * extract subdomain
	subdomain := name
	if zone != "." {
		if name == zone {
			subdomain = ""
		} else {
			subdomain = strings.TrimSuffix(name, "."+zone)
		}
	}

	// * prepare channel and send query
	no := r.no.Add(1)
	ch := make(chan *proto.ResolveResult, 1)
	r.pending.Store(no, ch)
	defer r.pending.Delete(no)

	client.Mutex.Lock()
	err := client.Stream.Send(&proto.ResolveQuery{
		No:        no,
		Type:      qtype,
		Zone:      zone,
		Subdomain: subdomain,
	})
	client.Mutex.Unlock()
	if err != nil {
		return nil, nil
	}

	select {
	case res := <-ch:
		return res, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled")
	}
}

func MapperResolveResult(res *proto.ResolveResult) *model.ResolveResult {
	now := time.Now()
	rcode := model.Rcode(res.Rcode)
	expiredAt := now.Add(time.Duration(res.Ttl) * time.Second)
	records := make([]*model.Record, len(res.Rrs))
	for i, rr := range res.Rrs {
		records[i] = &model.Record{
			Name:  &rr.Name,
			Type:  &rr.Type,
			Value: &rr.Value,
		}
	}

	return &model.ResolveResult{
		No:         &res.No,
		Rcode:      &rcode,
		ResolvedAt: &now,
		ExpiredAt:  &expiredAt,
		Records:    records,
	}
}
