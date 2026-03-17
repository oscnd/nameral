package dns

import (
	"sync"
	"sync/atomic"

	"go.scnd.dev/open/nameral/generate/proto"
	"google.golang.org/grpc"
)

type ClientStream struct {
	Name    string
	Stream  grpc.BidiStreamingServer[proto.ResolveResult, proto.ResolveQuery]
	mu      sync.Mutex
	no      atomic.Uint64
	pending sync.Map // uint64 → chan *proto.ResolveResult
}

func (r *ClientStream) Deliver(no uint64, result *proto.ResolveResult) {
	if ch, loaded := r.pending.LoadAndDelete(no); loaded {
		ch.(chan *proto.ResolveResult) <- result
	}
}
