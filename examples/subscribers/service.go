package main

import (
	"context"
	"fmt"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
)

func NewService(js nats.JetStreamContext) *UserService {
	return &UserService{
		js: js,
	}
}

type UserService struct {
	js nats.JetStreamContext
}

func (s UserService) Handle(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := NewAggregate(m.AggregateId)

	bee.Replay(ctx, s.js, m.Aggregate, m.AggregateId, bee.DeliverAll, agg)

	if agg.Found && m.CommandType == "create" {
		return nil, fmt.Errorf("aggregate %s with ID %s already exists", m.Aggregate, m.AggregateId)
	}

	if m.CommandType == "create" {
		return agg.ApplyCommand(ctx, m)
	}

	return agg.ApplyCommand(ctx, m)
}
