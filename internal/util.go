package internal

import (
	"context"
	"sync"
)

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

// --

type identityctxkey string

const identitykey = "identity"

func WithIdentity(ctx context.Context, identity GonnectIdentity) context.Context {
	return context.WithValue(ctx, identitykey, identity)
}

func IdentityFromContext(ctx context.Context) *GonnectIdentity {
	v := ctx.Value(wgkey)
	if v == nil {
		return nil
	}
	if identity, ok := v.(*GonnectIdentity); ok {
		return identity
	}

	return nil
}
