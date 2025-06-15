package bee

import (
	"context"

	"github.com/nats-io/nats.go"
)

type key[T any] struct{}

func with[T any](ctx context.Context, k key[T], v T) context.Context {
	return context.WithValue(ctx, k, v)
}

func get[T any](ctx context.Context, k key[T]) (T, bool) {
	v, ok := ctx.Value(k).(T)
	return v, ok
}

var natsKey = key[*nats.Conn]{}

func WithNats(ctx context.Context, nc *nats.Conn) context.Context {
	return with(ctx, natsKey, nc)
}
func Nats(ctx context.Context) (*nats.Conn, bool) {
	nc, ok := get(ctx, natsKey)
	if !ok {
		return nil, false
	}
	return nc, true
}

var jsKey = key[nats.JetStreamContext]{}

func WithJetStream(ctx context.Context, js nats.JetStreamContext) context.Context {
	return with(ctx, jsKey, js)
}
func JetStream(ctx context.Context) (nats.JetStreamContext, bool) {
	js, ok := get(ctx, jsKey)
	if !ok {
		return nil, false
	}
	return js, true
}
