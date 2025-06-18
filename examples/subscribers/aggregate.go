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

	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	switch event := event.(type) {
	case *UserCreated:
		u.Name = event.Name
		u.Country = event.Country
		u.Deleted = false
	case *UserUpdated:
		u.Country = event.Country
		u.Name = event.Name
	case *UserDeleted:
		u.Deleted = true
	}
	return nil
}

func (u *UserAggregate) ApplyCommand(_ context.Context, c *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	if c.AggregateId != u.ID {
		return nil, fmt.Errorf("aggregate ID mismatch")
	}

	_ = c.ExtraMetadata.AsMap()

	var event *gen.EventEnvelope = &gen.EventEnvelope{AggregateId: u.ID}
	event.AggregateType = "users"

	command, err := bee.UnmarshalCommand(c)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch command.(type) {
	case *CreateUserCommand:
		event.EventType = "created"
		event.Payload = c.Payload
	case *UpdateUserCommand:
		if u.Deleted {
			return nil, fmt.Errorf("cannot update deleted user")
		}
		event.EventType = "updated"
		event.Payload = c.Payload
	case *DeleteUserCommand:
		if u.Deleted {
			return nil, fmt.Errorf("user already deleted")
		}
		event.EventType = "deleted"
	default:
		return nil, fmt.Errorf("unknown command type: %s", c.CommandType)
	}
	return []*gen.EventEnvelope{event}, nil
}
