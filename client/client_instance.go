package client

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.scnd.dev/open/nameral/generate/proto"
	"go.scnd.dev/open/nameral/type/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Namera struct {
	config       *Config
	conn         *grpc.ClientConn
	handlers     *sync.Map // zone string → func(*model.HandleQuery) (*model.HandleResponse, error)
	mutex        sync.Mutex
	cancel       context.CancelFunc
	streamCancel context.CancelFunc
}

func (r *Namera) Handle(zone string, handle func(*model.HandleQuery) (*model.HandleResponse, error)) {
	r.handlers.Store(zone, handle)
	r.mutex.Lock()
	if r.streamCancel != nil {
		r.streamCancel()
	}
	r.mutex.Unlock()
}

func (r *Namera) Flush(zone string) error {
	r.handlers.Delete(zone)
	ctx := context.Background()
	_, err := proto.NewResolverClient(r.conn).Flush(ctx, &proto.FlushRequest{Zone: zone})
	return err
}

func (r *Namera) Close() error {
	r.cancel()
	return r.conn.Close()
}

func (r *Namera) stream(ctx context.Context) {
	for {
		if err := r.streamOnce(ctx); err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
				// retry after backoff
			}
		} else {
			select {
			case <-ctx.Done():
				return
			default:
				// reconnect
			}
		}
	}
}

func (r *Namera) streamOnce(parentCtx context.Context) error {
	// Per-stream context so Handle() can cancel just this stream.
	streamCtx, streamCancel := context.WithCancel(parentCtx)
	r.mutex.Lock()
	r.streamCancel = streamCancel
	r.mutex.Unlock()
	defer streamCancel()

	// Collect registered zones
	var zones []string
	r.handlers.Range(func(key, _ any) bool {
		zones = append(zones, key.(string))
		return true
	})

	if len(zones) == 0 {
		// No zones registered yet; wait before retrying
		select {
		case <-streamCtx.Done():
		case <-time.After(1 * time.Second):
		}
		return nil
	}

	// Build outgoing metadata
	md := metadata.New(map[string]string{
		"authorization": *r.config.Secret,
		"zones":         strings.Join(zones, ","),
	})
	metaCtx := metadata.NewOutgoingContext(streamCtx, md)

	stream, err := proto.NewResolverClient(r.conn).Resolve(metaCtx)
	if err != nil {
		if parentCtx.Err() != nil {
			return nil
		}
		return err
	}

	for {
		query, err := stream.Recv()
		if err != nil {
			// Cancelled by Handle() — reconnect immediately without backoff.
			if streamCtx.Err() != nil && parentCtx.Err() == nil {
				return nil
			}
			return err
		}
		go r.dispatch(query, stream)
	}
}

func (r *Namera) dispatch(query *proto.ResolveQuery, stream grpc.BidiStreamingClient[proto.ResolveResult, proto.ResolveQuery]) {
	handlerVal, ok := r.handlers.Load(query.Zone)
	if !ok {
		_ = stream.Send(&proto.ResolveResult{
			No:    query.No,
			Rcode: "NXDOMAIN",
		})
		return
	}

	handle := handlerVal.(func(*model.HandleQuery) (*model.HandleResponse, error))
	resp, err := handle(&model.HandleQuery{
		Type:      &query.Type,
		Zone:      &query.Zone,
		Subdomain: &query.Subdomain,
	})

	if err != nil || resp == nil {
		_ = stream.Send(&proto.ResolveResult{
			No:    query.No,
			Rcode: "SERVFAIL",
		})
		return
	}

	rcode := "NOERROR"
	if resp.Rcode != nil {
		rcode = *resp.Rcode
	}

	var rrs []*proto.RR
	for _, rec := range resp.Records {
		if rec.Name != nil && rec.Type != nil && rec.Value != nil {
			rrs = append(rrs, &proto.RR{
				Name:  *rec.Name,
				Type:  *rec.Type,
				Value: *rec.Value,
			})
		}
	}

	ttl := uint32(0)
	if resp.Ttl != nil {
		ttl = uint32(*resp.Ttl)
	}

	_ = stream.Send(&proto.ResolveResult{
		No:    query.No,
		Rcode: rcode,
		Ttl:   ttl,
		Rrs:   rrs,
	})
}
