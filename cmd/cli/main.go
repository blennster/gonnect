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
	device = flag.String("device", "", "device to operate on")
)

func main() {
	flag.Parse()
	client, err := rpc.DialHTTP("unix", "/tmp/gonnect.sock")
	if err != nil {
		slog.Error("failed to dial rpc, is your server running?", "err", err)
		os.Exit(1)
	}

	printUsage := func() {
		flag.PrintDefaults()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "pair":
		if *device != "" {
			var reply string
			fmt.Printf("pairing with %s\n", *device)
			err = client.Call("GonnectRpc.Pair", *device, &reply)
			if err != nil {
				panic(err)
			}

			fmt.Println(reply)
			return
		}
		printUsage()
	case "unpair":
		if *device != "" {
			var reply string
			fmt.Printf("unpairing with %s\n", *device)
			err = client.Call("GonnectRpc.Unpair", *device, &reply)
			if err != nil {
				panic(err)
			}

			fmt.Println(reply)
			return
		}
		printUsage()
	case "get":
		var reply []string
		err = client.Call("GonnectRpc.GetDevices", struct{}{}, &reply)
		if err != nil {
			panic(err)
		}

		for _, name := range reply {
			fmt.Println(name)
		}
		return
	}
}
