package resolveHandler

import (
	"go.scnd.dev/open/nameral/generate/proto"
	"google.golang.org/grpc"
)

func (h *Handler) Resolve(server grpc.BidiStreamingServer[proto.ResolveResult, proto.ResolveQuery]) error {
	return nil
}
