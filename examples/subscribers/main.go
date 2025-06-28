package main

import (
	"context"
	"runtime"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/co"
	"github.com/blinkinglight/bee/po"
	"github.com/blinkinglight/bee/qo"
	"github.com/nats-io/nats.go"
)

func main() {
	ctx := context.Background()

	nc, err := nats.Connect(nats.DefaultURL)

	if err != nil {
		panic(err)
	}

	js, err := nc.JetStream()

	if err != nil {
		panic(err)
	}

	ctx = bee.WithNats(ctx, nc)
	ctx = bee.WithJetStream(ctx, js)

	go bee.Command(ctx, NewService(ctx), co.WithAggreate("users"))

	go bee.Project(ctx, NewUserProjection(), po.WithAggreate("users"))

	go bee.Query(ctx, NewUserProjection(), qo.WithAggreate("users"))

	runtime.Goexit()
}
