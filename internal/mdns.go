package internal

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

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
			log.Println(entry)
			names = append(names, entry.AddrIPv4[0].String())
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	resolver.Browse(ctx, "_kdeconnect._udp", "local.", entries)

	<-ctx.Done()
	return names
}

func Announce(shutdown chan struct{}, wg *sync.WaitGroup) {
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

	log.Println("Published service:")
	log.Println("- Name:", name)
	log.Println("- Type:", service)
	log.Println("- Domain:", domain)
	log.Println("- Port:", port)

	<-shutdown
	log.Println("Shutting down.")
}
