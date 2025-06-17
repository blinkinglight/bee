package main

import (
	"context"
	"fmt"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"google.golang.org/protobuf/types/known/structpb"
)

func NewService() *UserService {
	return &UserService{}
}

type UserService struct {
}

func (s UserService) Handle(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := NewAggregate(m.AggregateId)

	bee.Replay(ctx, m.Aggregate, m.AggregateId, bee.DeliverAll, agg)

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

	return agg.ApplyCommand(ctx, m)
}
