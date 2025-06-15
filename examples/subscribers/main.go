package main

import (
	"context"
	"runtime"

	"github.com/blinkinglight/bee"
	"github.com/nats-io/nats.go"
)

func main() {
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

	go bee.NewCommandProcessor(context.Background(), nc, js)

	if err := bee.Register(context.Background(), "users", NewService(js).Handle); err != nil {
		panic(err)
	}

	runProjection(js)

	runtime.Goexit()
}
