package main

import (
	"context"
	"runtime"

	"github.com/blinkinglight/bee"
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

	js.AddStream(&nats.StreamConfig{
		Name:      "EVENTS",
		Subjects:  []string{"events.>"},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    0,
		Replicas:  1,
	})

	ctx = bee.WithNats(ctx, nc)
	ctx = bee.WithJetStream(ctx, js)

	go bee.NewCommandProcessor(ctx, "cmds.users", "users_cmd", NewService(js).Handle)

	runProjection(js)

	runtime.Goexit()
}
