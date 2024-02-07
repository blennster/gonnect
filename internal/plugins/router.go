package plugins

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/blennster/gonnect/internal"
)

type GonnectPlugin interface {
	React(context.Context, []byte) any
}

func Route(ctx context.Context, data []byte) []byte {
	var packet internal.GonnectPacket[any]
	err := json.Unmarshal(data, &packet)
	if err != nil {
		panic(err)
	}
	var plugin GonnectPlugin

	switch packet.Type {
	case internal.GonnectPingType:
		plugin = PingPlugin{}
	case internal.GonnectClipboardConnectType:
		slog.Debug("clipboard connection")
		return nil
	case internal.GonnectClipboardType:
		plugin = ctx.Value(internal.GonnectClipboardType).(ClipboardPlugin)
	default:
		slog.Error("unknown packet type", "type", packet.Type)
		return nil
	}

	if plugin == nil {
		panic("plugin is nil")
	}

	pkt := plugin.React(ctx, data)
	buf, err := json.Marshal(pkt)
	if err != nil {
		slog.Error("error marshalling packet for plugin", "plugin", plugin, "err", err)
		panic(err)
	}

	return buf
}
