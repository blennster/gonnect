package discover

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/blennster/gonnect/internal"
	"github.com/grandcat/zeroconf"
)

func GetDevices() []string {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		panic(err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	names := make([]string, 0)

	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			slog.Debug("MDNS", "Got entry", entry)
			names = append(names, entry.AddrIPv4[0].String())
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	resolver.Browse(ctx, "_kdeconnect._udp", "local.", entries)

	<-ctx.Done()
	return names
}

func AnnounceMdns(ctx context.Context) {
	wg := internal.WgFromContext(ctx)
	defer wg.Done()

	name := "gonnect"
	service := "_kdeconnect._udp"
	domain := "local."
	port := 1716
	hostname, _ := os.Hostname()
	// Setup our service export
	server, err := zeroconf.Register(name, service, domain, port,
		[]string{
			"type=desktop",
			"name=" + hostname,
			"id=" + name,
			"protocol=7",
		}, nil)
	defer server.Shutdown()

	if err != nil {
		panic(err)
	}

	slog.Info("published zeroconf service",
		slog.Group("service",
			"Name:", name,
			"Type:", service,
			"Domain:", domain,
			"Port:", port,
		),
	)

	<-ctx.Done()
	slog.Info("shutting down zeroconf.")
}
