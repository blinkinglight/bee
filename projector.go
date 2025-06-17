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

func Project(ctx context.Context, fn EventApplier, opts ...po.Options) error {

	cfg := &po.Config{
		AggreateID: "*",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.Aggreate == "" {
		panic("aggregate is required for projection")
	}

	if cfg.DurableName == "" {
		cfg.DurableName = cfg.Aggreate
	}

	if cfg.Subject == "" {
		cfg.Subject = fmt.Sprintf("events.%s.%s.>", cfg.Aggreate, cfg.AggreateID)
	}

	prefix := ""
	if cfg.Prefix != "" {
		prefix = cfg.Prefix + "_"
	}

	js, _ := JetStream(ctx)
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
