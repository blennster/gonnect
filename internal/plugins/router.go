package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/blennster/gonnect/internal"
)

func Handle(ctx context.Context, data []byte) []byte {
	var packet internal.GonnectPacket[any]
	err := json.Unmarshal(data, &packet)
	if err != nil {
		panic(err)
	}
	var plugin GonnectPlugin

	var t any
	switch packet.Type {
	case internal.GonnectPingType:
		t = ctx.Value(internal.GonnectPingType)
	case internal.GonnectClipboardType, internal.GonnectClipboardConnectType:
		t = ctx.Value(internal.GonnectClipboardType)
	default:
		slog.Error("unknown packet type in plugin handler", "type", packet.Type)
		return nil
	}

	plugin = t.(GonnectPlugin)

	if plugin == nil {
		panic(fmt.Sprintf("no plugin found for packet %q", data))
	}

	pkt := plugin.React(ctx, data)
	response, err := json.Marshal(pkt)
	if err != nil {
		slog.Error("error marshalling response from plugin", "plugin", plugin, "err", err)
		panic(err)
	}

	return response
}
