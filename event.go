package bee

import (
	"context"
	"fmt"
	"log"

	"github.com/blinkinglight/bee/eo"
	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type ManagerEventApplier interface {
	Handle(event *gen.EventEnvelope) ([]*gen.CommandEnvelope, error)
}

func Event(ctx context.Context, fn ManagerEventApplier, opts ...eo.Options) error {

	cfg := &eo.Config{
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
		cfg.Subject = fmt.Sprintf(EventsPrefix+".%s.%s.>", cfg.Aggregate, cfg.AggregateID)
	}

	prefix := ""
	if cfg.Prefix != "" {
		prefix = cfg.Prefix + "_"
	}

	js, _ := JetStream(ctx)

	js.AddStream(&nats.StreamConfig{
		Name:      "EVENTS",
		Subjects:  []string{EventsPrefix + ".>"},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    0,
		Replicas:  1,
	})

	sub, err := js.Subscribe(cfg.Subject, func(msg *nats.Msg) {
		if msg == nil {
			return
		}
		evt := &gen.EventEnvelope{}
		if err := proto.Unmarshal(msg.Data, evt); err != nil {
			msg.Ack()
			return
		}

		commands, err := fn.Handle(evt)
		if err != nil {
			msg.Ack()
			return
		}
		for _, command := range commands {
			if command.Aggregate == "" {
				command.Aggregate = evt.AggregateType
			}
			eventSubject := fmt.Sprintf(CommandsPrefix+".%s", command.Aggregate)
			b, _ := proto.Marshal(command)
			if _, err := js.Publish(eventSubject, b); err != nil {
				log.Printf("Error publishing event %v", err)
			}
		}

		_ = msg.Ack()
	}, nats.DeliverAll(), nats.ManualAck(), nats.Durable("procmgrs_"+prefix+cfg.DurableName), nats.BindStream("EVENTS"), nats.ConsumerName(cfg.DurableName))
	if err != nil {
		fmt.Printf("Error subscribing to events: %v\n", err)
		return err
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	return nil
}
