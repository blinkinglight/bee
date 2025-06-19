package bee

import (
	"context"
	"fmt"

	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/po"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type EventApplier interface {
	ApplyEvent(event *gen.EventEnvelope) error
}
type Querier interface {
	Query(query *gen.QueryEnvelope) (interface{}, error)
}

type Projector interface {
	EventApplier
	Querier
}

// Project subscribes to events for a specific aggregate and applies them using the provided EventApplier function.
// It uses JetStream to manage the event stream and durable subscriptions.
// The function takes a context, an EventApplier function, and optional configuration options.
// The configuration options allow customization of the aggregate type, aggregate ID, subject, durable name,
// and prefix for the subscription.
// po.WithSubject sets the subject for the subscription
// po.WithAggreate sets the aggregate type for the subscription
// po.WithAggrateID sets the aggregate ID for the subscription
// po.WithPrefix sets a prefix for the durable name
// po.WithDurable sets the durable name for the subscription
func Project(ctx context.Context, fn EventApplier, opts ...po.Options) error {

	cfg := &po.Config{
		AggregateID: "*",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.Aggregate == "" {
		panic("aggregate is required for projection")
	}

	if cfg.DurableName == "" {
		cfg.DurableName = cfg.Aggregate
	}

	if cfg.Subject == "" {
		cfg.Subject = fmt.Sprintf("events.%s.%s.>", cfg.Aggregate, cfg.AggregateID)
	}

	prefix := ""
	if cfg.Prefix != "" {
		prefix = cfg.Prefix + "_"
	}

	js, _ := JetStream(ctx)

	js.AddStream(&nats.StreamConfig{
		Name:      "EVENTS",
		Subjects:  []string{"events.>"},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    0,
		Replicas:  1,
	})

	sub, err := js.Subscribe(cfg.Subject, func(msg *nats.Msg) {
		if msg == nil {
			return
		}
		m := &gen.EventEnvelope{}
		if err := proto.Unmarshal(msg.Data, m); err != nil {
			msg.Ack()
			return
		}

		if err := fn.ApplyEvent(m); err != nil {
			msg.Ack()
			return
		}
		msg.Ack()
	}, nats.DeliverAll(), nats.ManualAck(), nats.Durable("events_"+prefix+cfg.DurableName), nats.BindStream("EVENTS"), nats.ConsumerName(cfg.DurableName))
	if err != nil {
		fmt.Printf("Error subscribing to events: %v\n", err)
		return err
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	return nil
}
