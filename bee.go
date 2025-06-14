package bee

import (
	"context"
	"fmt"
	"time"

	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// interface
type ReplayHandler interface {
	ApplyEvent(m *gen.EventEnvelope) error
}

func Replay(ctx context.Context, js nats.JetStreamContext, aggregate, id string, fn ReplayHandler) {
	lctx, cancel := context.WithCancel(ctx)
	msgs := make(chan *nats.Msg, 128)
	sub, err := js.ChanSubscribe(fmt.Sprintf("events.%s.%s.>", aggregate, id), msgs, nats.DeliverAll(), nats.ManualAck())
	if err != nil {
		cancel()
		return
	}
	defer close(msgs)
	defer sub.Unsubscribe()

	go func() {
		delay := 1000 * time.Millisecond
		waiter := time.NewTimer(delay)
		waiter.Reset(delay)
		for {
			select {
			case <-ctx.Done():
				cancel()
				return
			case <-waiter.C:
				cancel()
				return
			case msg := <-msgs:
				if msg == nil {
					continue
				}
				waiter.Reset(200 * time.Millisecond)
				var event = &gen.EventEnvelope{}
				_ = proto.Unmarshal(msg.Data, event)

				fn.ApplyEvent(event)
				msg.Ack()
			}
		}
	}()
	select {
	case <-lctx.Done():
	case <-ctx.Done():
	}
}
