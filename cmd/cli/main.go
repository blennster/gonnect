package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/blennster/gonnect/internal"
	"github.com/blennster/gonnect/internal/discover"
)

func setupLogger() {
	c := slog.HandlerOptions{}
	if os.Getenv("DEBUG") == "1" {
		c.Level = slog.LevelDebug
	}

	h := slog.NewTextHandler(os.Stdout, &c)
	slog.SetDefault(slog.New(h))
}

func main() {
	// client, _ := rpc.DialHTTP("unix", "gonnect.sock")
	// var reply string
	// client.Call("T.Hello", "World", &reply)
	// slog.Info(reply)
	// return

	setupLogger()

	shutdown := make(chan struct{})
	wg := sync.WaitGroup{}
	ctx := internal.WithWg(context.Background(), &wg)

	discover.Announce(ctx)

	// Wait for interrupt
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	// Closing a channel notifies all goroutines.
	close(shutdown)

	// This is just for synchronisation to make sure everything has been finished
	wg.Wait()

	slog.Info("Byebye")
}
