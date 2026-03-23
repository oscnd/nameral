package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/miekg/dns"
	"go.scnd.dev/open/nameral/client"
	"go.scnd.dev/open/nameral/command/agent/handler"
	recordEndpoint "go.scnd.dev/open/nameral/command/agent/handler/record"
	"go.scnd.dev/open/nameral/module/resolve"
	"go.scnd.dev/open/nameral/module/store"
	"go.scnd.dev/open/polygon/compat/common"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			fx.Annotate(
				common.Config[Config],
				fx.As(new(common.PolygonConfig)),
				fx.As(new(handler.Config)),
			),
			common.Config[Config],
			common.Polygon,
			provideRecordStore,
			provideFiber,
			recordEndpoint.Handle,
		),
		fx.Invoke(
			handler.Bind,
			invoke,
		),
	).Run()
}

func provideRecordStore(config *Config) *store.Store {
	if config.RecordFile == nil {
		return nil
	}
	s := store.NewStore(*config.RecordFile)
	s.Load()
	return s
}

func provideFiber(lc fx.Lifecycle, config *Config) *fiber.App {
	if config.RecordFile == nil || config.RecordKey == nil || len(config.WebListen) == 0 {
		return nil
	}
	app := fiber.New(fiber.Config{ErrorHandler: common.FiberError})
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			for _, addr := range config.WebListen {
				a := *addr
				go func() {
					_ = app.Listen(a)
				}()
			}
			return nil
		},
		OnStop: func(_ context.Context) error {
			return app.Shutdown()
		},
	})
	return app
}

func invoke(lc fx.Lifecycle, config *Config, store *store.Store) error {
	var tlsConfig *tls.Config
	if config.CertificateFile != nil {
		pem, err := os.ReadFile(*config.CertificateFile)
		if err != nil {
			return err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		tlsConfig = &tls.Config{RootCAs: pool}
	}

	namera, err := client.New(&client.Config{
		Addresses: config.Addresses,
		Secret:    config.Secret,
		Tls:       tlsConfig,
	})
	if err != nil {
		return err
	}

	if store != nil {
		go store.Tick()
	}

	lookup := &resolve.Resolve{
		Store:     store,
		DnsClient: &dns.Client{},
		Upstream:  config.Upstream,
	}

	for _, zone := range config.Zones {
		z := *zone
		namera.Handle(z, lookup.Handle)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			if store != nil {
				store.Stop()
			}
			return namera.Close()
		},
	})

	return nil
}
