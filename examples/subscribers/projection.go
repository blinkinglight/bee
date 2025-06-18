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

func (up UserProjection) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		log.Printf("error unmarshalling event: %v", err)
	}
	switch event := event.(type) {
	case *UserCreated:
		println("User created:", event.Name, "from", event.Country)
	case *UserUpdated:
		println("User updated:", event.Name, "from", event.Country)
	case *UserDeleted:
		println("User deleted with ID:", e.AggregateId)
	default:
		println("Unknown event type:", e.EventType)
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
