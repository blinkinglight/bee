package main

import (
	"context"
	"fmt"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
	"google.golang.org/protobuf/types/known/structpb"
)

func NewService(ctx context.Context) *UserService {
	return &UserService{ctx: ctx}
}

type UserService struct {
	ctx context.Context
}

func (s UserService) Handle(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := NewAggregate(m.AggregateId)

	bee.Replay(s.ctx, agg, ro.WithAggreate(m.Aggregate), ro.WithAggregateID(m.AggregateId))

	if agg.Found && m.CommandType == "create" {
		return nil, fmt.Errorf("aggregate %s with ID %s already exists", m.Aggregate, m.AggregateId)
	}

	m.ExtraMetadata, _ = structpb.NewStruct(map[string]any{
		"foo":   1,
		"bar":   "baz",
		"baz":   []any{"a", "b", "c"},
		"qux":   map[string]any{"key1": "value1", "key2": 2},
		"quux":  true,
		"quuz":  3.14,
		"corge": nil,
	})

	return agg.ApplyCommand(s.ctx, m)
}
