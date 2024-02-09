package plugins

import (
	"context"

	"github.com/blennster/gonnect/internal"
)

type GonnectPlugin interface {
	React(context.Context, []byte) any
}

type GonnectPluginMessage internal.ChanMsg

func WithPlugins(ctx context.Context) (context.Context, <-chan GonnectPluginMessage) {
	ch := make(chan GonnectPluginMessage)

	// ping plugin is stateless and non-bidirectional as of now
	ctx = context.WithValue(ctx, internal.GonnectPingType, pingPlugin{})

	c := NewClipboardPlugin(ctx, ch)
	ctx = context.WithValue(ctx, internal.GonnectClipboardType, c)

	return ctx, ch
}
