package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/rpc"
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
	} else {
		c.Level = slog.LevelInfo
	}

	h := slog.NewTextHandler(os.Stdout, &c)
	slog.SetDefault(slog.New(h))
	slog.Debug("Enabling debug logging")
}

func setupRpc(t any) net.Listener {
	rpc.Register(t)
	rpc.HandleHTTP()
	l, err := net.Listen("unix", "/tmp/gonnect.sock")
	if err != nil {
		panic(err)
	}
	go http.Serve(l, nil)

	return l
}

func main() {
	setupLogger()

	ctx, t := internal.WithRpc(context.Background())
	ctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}
	ctx = internal.WithWg(ctx, &wg)

	l := setupRpc(t)
	defer l.Close()
	defer os.Remove("/tmp/gonnect.sock")

	discover.Announce(ctx)
	// plugins.WatchClipboard(ctx)

	// Wait for interrupt
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	cancel()

	// This is just for synchronisation to make sure everything has been finished
	wg.Wait()

	slog.Info("Byebye")
}
