package bee

import (
	"context"
	"fmt"

	"github.com/blinkinglight/bee/gen"
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

func Project(ctx context.Context, durable, aggregate, id string, fn EventApplier) {
	js, _ := JetStream(ctx)
	sub, err := js.Subscribe(fmt.Sprintf("events.%s.%s.>", aggregate, id), func(msg *nats.Msg) {
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
	}, nats.DeliverAll(), nats.ManualAck(), nats.Durable("events_"+aggregate+"_"+durable), nats.BindStream("EVENTS"), nats.ConsumerName(durable))
	if err != nil {
		fmt.Printf("Error subscribing to events: %v\n", err)
		return
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
}
