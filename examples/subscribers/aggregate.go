package main

import (
	"context"
	"fmt"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
)

// --- UserAggregate implements ES aggregate logic ---
type UserAggregate struct {
	ID      string
	Name    string
	Country string
	Deleted bool

	Found bool // Used to check if the aggregate was found during replay
}

type User struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

func NewAggregate(id string) *UserAggregate {
	return &UserAggregate{ID: id}
}

func (u *UserAggregate) ApplyEvent(e *gen.EventEnvelope) error {
	u.Found = true
	switch e.EventType {
	case "created":
		data, _ := bee.Unmarshal[User](e.Payload)
		u.Name = data.Name
		u.Country = data.Country
		u.Deleted = false
	case "updated":
		data, _ := bee.Unmarshal[User](e.Payload)
		u.Country = data.Country
		u.Name = data.Name
	case "deleted":
		u.Deleted = true
	}
	return nil
}

func (u *UserAggregate) ApplyCommand(_ context.Context, c *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	if c.AggregateId != u.ID {
		return nil, fmt.Errorf("aggregate ID mismatch")
	}
	var event *gen.EventEnvelope = &gen.EventEnvelope{AggregateId: u.ID}
	event.AggregateType = "users"
	switch c.CommandType {
	case "create":
		event.EventType = "created"
		event.Payload = c.Payload
	case "update":
		if u.Deleted {
			return nil, fmt.Errorf("cannot update deleted user")
		}
		event.EventType = "updated"
		event.Payload = c.Payload
	case "delete":
		if u.Deleted {
			return nil, fmt.Errorf("user already deleted")
		}
		event.EventType = "deleted"
	default:
		return nil, fmt.Errorf("unknown command type: %s", c.CommandType)
	}
	return []*gen.EventEnvelope{event}, nil
}
