package resolveHandler

import (
	"context"
	"fmt"

	"go.scnd.dev/open/nameral/common/config"
	commonGrpc "go.scnd.dev/open/nameral/common/grpc"
	"go.scnd.dev/open/nameral/generate/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) Flush(ctx context.Context, req *proto.FlushRequest) (*proto.FlushResponse, error) {
	// Get pre-authenticated client from context
	clientConfig, ok := ctx.Value(commonGrpc.ClientContextKey).(*config.Client)
	if !ok || clientConfig == nil {
		return nil, status.Error(codes.Unauthenticated, "client not authenticated")
	}

	// Validate zone is within client's AllowedZones
	allowed := false
	for _, z := range clientConfig.AllowedZones {
		if *z == req.Zone {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "zone not allowed")
	}

	// Delete Redis keys matching the zone
	var pattern string
	if req.Zone == "." {
		pattern = "dns:*"
	} else {
		pattern = fmt.Sprintf("dns:*%s:*", req.Zone)
	}

	var cursor uint64
	for {
		keys, nextCursor, err := h.Redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		if len(keys) > 0 {
			h.Redis.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return &proto.FlushResponse{Ok: true}, nil
}
