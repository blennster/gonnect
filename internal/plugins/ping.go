package plugins

import (
	"context"
	"encoding/json"

	"github.com/blennster/gonnect/internal"
)

// Respond to ping messages from other device and ping the other device
type PingPlugin struct{}

var message string = "pong"

func (PingPlugin) React(ctx context.Context, data []byte) any {
	var packet internal.GonnectPacket[internal.GonnectPing]
	err := json.Unmarshal(data, &packet)
	if err != nil {
		panic(err)
	}

	pkt := internal.NewGonnectPacket[internal.GonnectPing](internal.GonnectPing{Message: &message})

	return pkt
}
