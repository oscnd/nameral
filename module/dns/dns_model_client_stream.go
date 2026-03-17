package dns

import (
	"sync"

	"go.scnd.dev/open/nameral/generate/proto"
	"google.golang.org/grpc"
)

type ClientStream struct {
	Name   string
	Dns    *Module
	Stream grpc.BidiStreamingServer[proto.ResolveResult, proto.ResolveQuery]
	Mutex  sync.Mutex
}

func (r *ClientStream) Deliver(no uint64, result *proto.ResolveResult) {
	if ch, loaded := r.Dns.pending.LoadAndDelete(no); loaded {
		ch.(chan *proto.ResolveResult) <- result
	}
}
