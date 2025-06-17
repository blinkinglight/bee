package bee

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const DeliverAll = 0

// interface
type ReplayHandler interface {
	ApplyEvent(m *gen.EventEnvelope) error
}

func Replay(ctx context.Context, aggregate, id string, seq uint64, fn ReplayHandler) {
	lctx, cancel := context.WithCancel(ctx)
	js, _ := JetStream(ctx)

	oneMsg := make(chan *nats.Msg, 1)
	ls, err := js.Subscribe(fmt.Sprintf("events.%s.%s.>", aggregate, id), func(msg *nats.Msg) {
		oneMsg <- msg
	}, nats.DeliverLast())

	if err != nil {
		cancel()
		return
	}
	if err := ls.AutoUnsubscribe(1); err != nil {
		cancel()
		return
	}
	c := time.NewTimer(50 * time.Millisecond)
	var lmsg *nats.Msg
	select {
	case <-c.C:
		cancel()
		return
	case <-ctx.Done():
		cancel()
		return
	case lmsg = <-oneMsg:
	}
	meta, _ := lmsg.Metadata()

	msgs := make(chan *nats.Msg, 128)
	opt := nats.DeliverAll()
	if seq > 0 {
		opt = nats.StartSequence(seq)
	}
	sub, err := js.Subscribe(fmt.Sprintf("events.%s.%s.>", aggregate, id), func(msg *nats.Msg) {
		msgs <- msg
	}, opt, nats.ManualAck())
	if err != nil {
		log.Printf("Replay: Error subscribing to %s.%s: %v", aggregate, id, err)
		cancel()
		return
	}

	defer sub.Unsubscribe()
	defer close(msgs)

	go func() {
		for {
			select {
			case <-ctx.Done():
				cancel()
				return
			case msg := <-msgs:
				if msg == nil {
					continue
				}
				m, _ := msg.Metadata()
				last := false
				if m.Sequence.Stream == meta.Sequence.Stream {
					last = true
				}
				var event = &gen.EventEnvelope{}
				if err := proto.Unmarshal(msg.Data, event); err != nil {
					_ = err
				}

				fn.ApplyEvent(event)
				msg.Ack()
				if last {
					cancel()
					return
				}
			}
		}
	}()
	select {
	case <-ctx.Done():
	case <-lctx.Done():
	}
}

func ReplayAndSubscribe[T EventApplier](ctx context.Context, aggregate, id string, agg T) <-chan T {
	ch := make(chan T, 128)
	msgs := make(chan *nats.Msg, 128)
	js, _ := JetStream(ctx)
	log.Printf("ReplayAndSubscribe: Received message for %s.%s", aggregate, id)
	sub, err := js.Subscribe(fmt.Sprintf("events.%s.%s.>", aggregate, id), func(msg *nats.Msg) {
		msgs <- msg
	}, nats.ManualAck(), nats.DeliverNew())
	if err != nil {
		log.Printf("ReplayAndSubscribe: Error subscribing to %s.%s: %v", aggregate, id, err)
		ch <- agg
		close(ch)
		return ch
	}
	Replay(ctx, aggregate, id, DeliverAll, agg)
	ch <- agg
	go func() {
		defer sub.Unsubscribe()
		defer close(ch)
		defer close(msgs)
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgs:
				event := &gen.EventEnvelope{}
				if err := proto.Unmarshal(msg.Data, event); err != nil {
					msg.Respond([]byte(err.Error()))
					msg.Ack()
					continue
				}
				agg.ApplyEvent(event)
				ch <- agg
				msg.Ack()
			}
		}
	}()
	return ch
}
