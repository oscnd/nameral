package dns

import (
	"context"

	"github.com/bsthun/gut"
	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/common/config"
	"go.uber.org/fx"
)

func Init(lifecycle fx.Lifecycle, config *config.Config) *dns.Server {
	// * initialize dns server
	server := &dns.Server{Addr: *config.DnsListen, Net: "udp"}

	// * setup lifecycle
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := server.ListenAndServe(); err != nil {
					gut.Fatal("failed to start server", err)
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			_ = server.Shutdown()
			return nil
		},
	})

	return server
}
