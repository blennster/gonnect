package plugins

import (
	"context"

	"github.com/blennster/gonnect/internal"
)

// func Init(ctx context.Context) <-chan []byte {
// 	ch := make(chan []byte)
//
// 	go clipboardWatcher(ctx, ch)
//
// 	return ch
// }

func WithPlugins(ctx context.Context) (context.Context, <-chan []byte) {
	ch := make(chan []byte)
	c := NewClipboardPlugin(ctx, ch)
	ctx = context.WithValue(ctx, internal.GonnectClipboardType, c)

	return ctx, ch
}
