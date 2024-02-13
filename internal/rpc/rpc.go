package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/blennster/gonnect/internal/discover"
	"github.com/blennster/gonnect/internal/security"
)

type rpcctxkey string

const rpckey = rpcctxkey("rpc")

type GonnectRpc struct {
}

func (*GonnectRpc) Hello(msg string, reply *string) error {
	*reply = "Hello " + msg
	fmt.Println(msg)
	return nil
}

func (*GonnectRpc) Pair(deviceid string, reply *string) error {
	slog.Info("rpc pair request", "device", deviceid)

	if security.Devices.Get(deviceid) != nil {
		*reply = fmt.Sprintf("device %q is already paired", deviceid)
		return nil
	}

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("pairing timed out")
		default:
			// Keep trying
			if security.ApprovePair(deviceid) {
				*reply = fmt.Sprintf("approved pairing with %q", deviceid)
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (*GonnectRpc) Unpair(deviceid string, reply *string) error {
	slog.Info("rpc unpair request", "device", deviceid)
	security.Devices.Remove(deviceid)
	*reply = fmt.Sprintf("unpaired with %q", deviceid)
	return nil
}

func (*GonnectRpc) GetDevices(_ struct{}, reply *[]string) error {
	r, err := discover.GetDevices()
	if err != nil {
		return err
	}
	*reply = r
	return nil
}

type Unsubscribe func()

func WithRpc(ctx context.Context) (context.Context, *GonnectRpc) {
	t := new(GonnectRpc)
	return context.WithValue(ctx, rpckey, t), t
}

func GetRpc(ctx context.Context) *GonnectRpc {
	return ctx.Value(rpckey).(*GonnectRpc)
}
