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
	} else {
		c.Level = slog.LevelInfo
	}

	h := slog.NewTextHandler(os.Stdout, &c)
	slog.SetDefault(slog.New(h))
	slog.Debug("Enabling debug logging")
}

func main() {
	// t := new(internal.T)
	// rpc.Register(t)
	// rpc.HandleHTTP()
	// l, err := net.Listen("unix", "gonnect.sock")
	// if err != nil {
	// 	panic(err)
	// }
	// defer l.Close()
	//
	// go http.Serve(l, nil)
	//
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// <-c

	setupLogger()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	ctx = internal.WithWg(ctx, &wg)

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
