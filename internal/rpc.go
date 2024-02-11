package internal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/blennster/gonnect/internal/security"
)

type rpcctxkey string

const rpckey = rpcctxkey("rpc")

type GonnectRpc struct {
	pairingBroker *Broker[string]
	replyCh       chan string
}

func (*GonnectRpc) Hello(msg string, reply *string) error {
	*reply = "Hello " + msg
	fmt.Println(msg)
	return nil
}

func (r *GonnectRpc) Pair(deviceid string, reply *string) error {
	slog.Info("rpc pair request", "device", deviceid)

	if security.Devices.Get(deviceid) != nil {
		*reply = fmt.Sprintf("device %q is already paired", deviceid)
		return nil
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout")
		case resp := <-r.replyCh:
			if resp == deviceid {
				*reply = fmt.Sprintf("paired with %q", deviceid)
				return nil
			}
			return fmt.Errorf("pairing failed %s", resp)
		default:
			// Keep trying
			r.pairingBroker.Publish(deviceid)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (r *GonnectRpc) Unpair(deviceid string, reply *string) error {
	slog.Info("rpc unpair request", "device", deviceid)
	security.Devices.Remove(deviceid)
	*reply = fmt.Sprintf("unpaired with %q", deviceid)
	return nil
}

type Unsubscribe func()

func (r *GonnectRpc) SubscribeToPair() (<-chan string, Unsubscribe) {
	ch := r.pairingBroker.Subscribe()
	return ch, func() { r.pairingBroker.Unsubscribe(ch) }
}

// Reply sends a reply to the rpc caller
// Note: this uses and ubuffered channel and will be blocking
func (r *GonnectRpc) Reply(message string) {
	r.replyCh <- message
}

func WithRpc(ctx context.Context) (context.Context, *GonnectRpc) {
	t := new(GonnectRpc)
	t.pairingBroker = NewBroker[string]()
	t.replyCh = make(chan string)
	go t.pairingBroker.Start()

	return context.WithValue(ctx, rpckey, t), t
}

func GetRpc(ctx context.Context) *GonnectRpc {
	return ctx.Value(rpckey).(*GonnectRpc)
}
