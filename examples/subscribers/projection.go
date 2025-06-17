package main

import (
	"log"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
)

func NewUserProjection() *UserProjection {
	return &UserProjection{}
}

type UserProjection struct {
}

func (up UserProjection) ApplyEvent(event *gen.EventEnvelope) error {
	log.Printf("got event on subject %s %s %s", event.AggregateType, event.AggregateId, event.Payload)
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
	return nil
}

func (p UserProjection) Query(query *gen.QueryEnvelope) (any, error) {
	log.Printf("got query on subject %s %s", query.QueryType, query.ExtraMetadata.AsMap())
	switch query.QueryType {
	case "one":
		return "query one fake result", nil
	case "many":
		return "query many fake result", nil
	case "any":
		return "query any fake result", nil
	default:
		return nil, nil
	}
}

// func runProjection(js nats.JetStreamContext) {
// 	// projection for now to console
// 	js.Subscribe("events.users.>", func(m *nats.Msg) {
// 		var event gen.EventEnvelope
// 		proto.Unmarshal(m.Data, &event)
// 		log.Printf("got event on subject %s %s", m.Subject, event.Payload)

// 		switch event.EventType {
// 		case "created":
// 			user, _ := bee.Unmarshal[User](event.Payload)
// 			println("User created:", user.Name, "from", user.Country)
// 		case "updated":
// 			user, _ := bee.Unmarshal[User](event.Payload)
// 			println("User updated:", user.Name, "from", user.Country)
// 		case "deleted":
// 			println("User deleted with ID:", event.AggregateId)
// 		default:
// 			println("Unknown event type:", event.EventType)
// 		}

// 	}, nats.Durable("vienas"))
// }
