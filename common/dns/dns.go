package dns

import (
	"context"

	"github.com/bsthun/gut"
	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/common/config"
	"go.uber.org/fx"
)

func Init(lifecycle fx.Lifecycle, config *config.Config) (*dns.Server, *dns.Server) {
	// * initialize udp server
	udpServer := &dns.Server{Addr: *config.DnsListen, Net: "udp"}

	// * initialize tcp server
	tcpServer := &dns.Server{Addr: *config.DnsListen, Net: "tcp"}

	// * setup lifecycle
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := udpServer.ListenAndServe(); err != nil {
					gut.Fatal("failed to start UDP server", err)
				}
			}()
			go func() {
				if err := tcpServer.ListenAndServe(); err != nil {
					gut.Fatal("failed to start TCP server", err)
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			_ = udpServer.Shutdown()
			_ = tcpServer.Shutdown()
			return nil
		},
	})

	return udpServer, tcpServer
}
