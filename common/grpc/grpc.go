package grpc

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/bsthun/gut"
	"go.scnd.dev/open/nameral/common/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Init(lc fx.Lifecycle, config *config.Config) *grpc.Server {
	// * Initialize interceptor
	interceptor := NewInterceptor(config)

	// * Initialize TLS with GetCertificate for hot-reload on cert rotation
	tlsConfig := &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(*config.ServerCertificateFile, *config.ServerPrivateKeyFile)
			return &cert, err
		},
	}

	// * Initialize gRPC server
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.UnaryInterceptor(interceptor.AuthorizationUnaryInterceptor),
		grpc.StreamInterceptor(interceptor.AuthorizationStreamInterceptor),
	)

	// * Append lifecycle hook
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				lis, err := net.Listen(*config.ProtoListen[0], *config.ProtoListen[1])
				if err != nil {
					gut.Fatal("Unable to listen", err)
				}
				if err := grpcServer.Serve(lis); err != nil {
					gut.Fatal("Unable to serve", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			grpcServer.GracefulStop()
			return nil
		},
	})

	// * Return gRPC server
	return grpcServer
}
