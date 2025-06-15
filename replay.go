package bee

import (
	"context"
	"fmt"

	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const DeliverAll = 0

// interface
type ReplayHandler interface {
	ApplyEvent(m *gen.EventEnvelope) error
}

func Replay(ctx context.Context, js nats.JetStreamContext, aggregate, id string, seq uint64, fn ReplayHandler) {
	lctx, cancel := context.WithCancel(ctx)

	ls, err := js.SubscribeSync(fmt.Sprintf("events.%s.%s.>", aggregate, id), nats.DeliverLast())
	if err != nil {
		cancel()
		return
	}
	ls.AutoUnsubscribe(1)
	lmsg, _ := ls.NextMsg(0)
	if lmsg == nil {
		cancel()
		return
	}

	meta, _ := lmsg.Metadata()

	msgs := make(chan *nats.Msg, 128)
	opt := nats.DeliverAll()
	if seq > 0 {
		opt = nats.StartSequence(seq)
	}
	sub, err := js.ChanSubscribe(fmt.Sprintf("events.%s.%s.>", aggregate, id), msgs, opt, nats.ManualAck())
	if err != nil {
		cancel()
		return
	}
	defer close(msgs)
	defer sub.Unsubscribe()

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
	case <-lctx.Done():
	case <-ctx.Done():
	}
}
