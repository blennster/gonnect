package internal

import (
	"context"
	"fmt"
	"log/slog"
)

type rpcctxkey string

const rpckey = rpcctxkey("rpc")

type GonnectRpc struct {
	PairCh chan string
}

func (*GonnectRpc) Hello(msg string, reply *string) error {
	*reply = "Hello " + msg
	fmt.Println(msg)
	return nil
}

func (r *GonnectRpc) Pair(deviceid string, reply *string) error {
	slog.Info("rpc pair", "with", deviceid)
	r.PairCh <- deviceid
	if resp := <-r.PairCh; resp != "ok" {
		// We read or own message
		if resp == deviceid {
			*reply = "there was no pair to accept"
			return nil
		}
		return fmt.Errorf("pairing failed %s", resp)
	}
	*reply = "paried with " + deviceid
	return nil
}

func WithRpc(ctx context.Context) (context.Context, *GonnectRpc) {
	t := new(GonnectRpc)
	t.PairCh = make(chan string, 1)
	return context.WithValue(ctx, rpckey, t), t
}

func GetRpc(ctx context.Context) *GonnectRpc {
	return ctx.Value(rpckey).(*GonnectRpc)
}
