package grpc

import (
	"context"

	"go.scnd.dev/open/nameral/common/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const ClientContextKey contextKey = "client"

func NewInterceptor(config *config.Config) *Interceptor {
	return &Interceptor{Config: config}
}

type Interceptor struct {
	Config *config.Config
}

func (r *Interceptor) AuthorizationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	client, err := r.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	return handler(context.WithValue(ctx, ClientContextKey, client), req)
}

func (r *Interceptor) AuthorizationStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	client, err := r.authenticate(ss.Context())
	if err != nil {
		return err
	}
	return handler(srv, &WrappedStream{ss, context.WithValue(ss.Context(), ClientContextKey, client)})
}

func (r *Interceptor) authenticate(ctx context.Context) (*config.Client, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	tokens := md["authorization"]
	if len(tokens) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization token not provided")
	}
	for _, c := range r.Config.Clients {
		if *c.Token == tokens[0] {
			return c, nil
		}
	}
	return nil, status.Error(codes.Unauthenticated, "invalid token")
}
