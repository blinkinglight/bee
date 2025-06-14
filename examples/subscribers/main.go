package main

import (
	"context"
	"log"
	"runtime"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
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

	go func() {
		bee.NewCommandProcessor(context.Background(), nc, js)
	}()

	if err := bee.Register(context.Background(), "users", NewService(js).Handle); err != nil {
		panic(err)
	}

	// projection for now to console
	js.Subscribe("events.users.>", func(m *nats.Msg) {
		var event gen.EventEnvelope
		proto.Unmarshal(m.Data, &event)
		log.Printf("got event on subject %s %s", m.Subject, event.Payload)

		switch event.EventType {
		case "created":
			user, _ := bee.Unmarshal[User](event.Payload)
			println("User created:", user.Name, "from", user.Country)
		case "updated":
			user, _ := bee.Unmarshal[User](event.Payload)
			println("User updated:", user.Name, "from", user.Country)
		case "deleted":
			println("User deleted with ID:", event.AggregateId)
		default:
			println("Unknown event type:", event.EventType)
		}

	}, nats.Durable("vienas"))

	runtime.Goexit()
}
