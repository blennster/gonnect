package plugins

import (
	"context"

	"github.com/blennster/gonnect/internal"
)

type GonnectPlugin interface {
	React(context.Context, []byte) any
}

var (
	_ GonnectPlugin = (*pingPlugin)(nil)
	_ GonnectPlugin = (*clipboardPlugin)(nil)
)

type GonnectPluginMessage internal.ChanMsg

func WithPlugins(ctx context.Context) (c context.Context, pluginCh <-chan GonnectPluginMessage) {
	ch := make(chan GonnectPluginMessage, 5)

	// ping plugin is stateless and non-bidirectional as of now
	ctx = context.WithValue(ctx, internal.GonnectPingType, pingPlugin{})

	cp := NewClipboardPlugin(ctx, ch)
	ctx = context.WithValue(ctx, internal.GonnectClipboardType, cp)

	return ctx, ch
}
