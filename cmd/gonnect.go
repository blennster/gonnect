package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/blennster/gonnect/internal"
)

func borrowWg(wg *sync.WaitGroup) *sync.WaitGroup {
	wg.Add(1)
	return wg
}

func main() {
	log.Println("Getting some devices")
	devs := internal.GetDevices()
	for _, dev := range devs {
		log.Println(dev)
	}

	shutdown := make(chan struct{})
	wg := sync.WaitGroup{}

	go internal.Announce(shutdown, borrowWg(&wg))
	go internal.ListenUdp(shutdown, borrowWg(&wg))
	// go internal.ListenTcp(shutdown, borrowWg(&wg))

	// Wait time zero is basically for ever
	waitTime := 0
	// Clean exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	// Timeout timer.
	var tc <-chan time.Time
	if waitTime > 0 {
		tc = time.After(time.Second * time.Duration(waitTime))
	}

	select {
	case <-sig:
		// Exit by user
	case <-tc:
		// Exit by timeout
	}

	// Closing a channel always sends a zero value
	close(shutdown)
	// This is just for synchronisation to make sure everything has been finished
	wg.Wait()

	log.Println("Byebye")
}
