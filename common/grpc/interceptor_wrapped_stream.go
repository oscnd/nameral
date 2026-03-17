package grpc

import (
	"context"

	"google.golang.org/grpc"
)

type WrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (r *WrappedStream) Context() context.Context {
	return r.ctx
}
