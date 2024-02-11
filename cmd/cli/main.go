package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/rpc"
	"os"
)

func setupLogger() {
	c := slog.HandlerOptions{}
	if os.Getenv("DEBUG") == "1" {
		c.Level = slog.LevelDebug
	}

	h := slog.NewTextHandler(os.Stdout, &c)
	slog.SetDefault(slog.New(h))
}

var (
	pair = flag.String("pair", "", "pair with device")
)

func main() {
	flag.Parse()
	client, err := rpc.DialHTTP("unix", "/tmp/gonnect.sock")
	if err != nil {
		slog.Error("failed to dial rpc, is your server running?", "err", err)
		os.Exit(1)
	}

	if *pair != "" {
		var reply string
		fmt.Printf("pairing with %s\n", *pair)
		err = client.Call("GonnectRpc.Pair", *pair, &reply)
		if err != nil {
			panic(err)
		}

		fmt.Println(reply)
		return
	}

	flag.PrintDefaults()
	os.Exit(1)

	var reply string
	// err = client.Call("GonnectRpc.Hello", "world", &reply)
	// if err != nil {
	// 	panic(err)
	// }
	// slog.Info(reply)
	err = client.Call("GonnectRpc.Pair", "d8261e07215dbc42", &reply)
	if err != nil {
		panic(err)
	}
	slog.Info(reply)
	return

	// setupLogger()
	//
	// shutdown := make(chan struct{})
	// wg := sync.WaitGroup{}
	// ctx := internal.WithWg(context.Background(), &wg)
	//
	// discover.Announce(ctx)
	//
	// // Wait for interrupt
	// sig := make(chan os.Signal, 1)
	// signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	// <-sig
	//
	// // Closing a channel notifies all goroutines.
	// close(shutdown)
	//
	// // This is just for synchronisation to make sure everything has been finished
	// wg.Wait()
	//
	// slog.Info("Byebye")
}
