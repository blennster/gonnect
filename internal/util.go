package internal

import (
	"context"
	"sync"
)

func BorrowWg(wg *sync.WaitGroup) *sync.WaitGroup {
	wg.Add(1)
	return wg
}

type wgctxkey string

const wgkey = "waitgroup"

func WithWg(ctx context.Context, wg *sync.WaitGroup) context.Context {
	return context.WithValue(ctx, wgkey, wg)
}

func WgFromContext(ctx context.Context) *sync.WaitGroup {
	v := ctx.Value(wgkey)
	if v == nil {
		return nil
	}
	if wg, ok := v.(*sync.WaitGroup); ok {
		wg.Add(1)
		return wg
	}

	return nil
}
