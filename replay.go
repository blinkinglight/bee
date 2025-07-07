package bee

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const DeliverAll = 0

// interface
type ReplayHandler interface {
	ApplyEvent(m *gen.EventEnvelope) error
}

// Replay replays events for a given aggregate and aggregate ID.
// accepts ro.Options to configure the replay behavior.
// ro.WithAggregate ro.WithAggregateID - configure the aggregate and aggregate ID
// ro.WithSubject - use custom subject instead of default "events.aggregate.aggregateID.>"
// ro.WithStartSeq - start from event (if you have snapshot)
// ro.WtihParent - nests subjects
// ro.WithTimeout - timeout if no events for stream
func Replay(ctx context.Context, fn ReplayHandler, opts ...ro.Options) error {

	cfg := &ro.Config{
		StartSeq: DeliverAll,
		Timeout:  50 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	subject := fmt.Sprintf("%s.%s.%s.>", EventsPrefix, cfg.Aggregate, cfg.AggregateID)
	if len(cfg.Parents) > 0 {
		cfg.Subject = fmt.Sprintf("%s.%s.%s.%s.>", EventsPrefix, strings.Join(cfg.Parents, "."), cfg.Aggregate, cfg.AggregateID)
	}

	if cfg.Subject != "" {
		subject = cfg.Subject
	}

	lctx, cancel := context.WithCancel(ctx)
	js, _ := JetStream(ctx)

	msgs := make(chan *nats.Msg, 128)
	opt := nats.DeliverAll()
	if cfg.StartSeq > 0 {
		opt = nats.StartSequence(cfg.StartSeq)
	}
	sub, err := js.Subscribe(subject, func(msg *nats.Msg) {
		select {
		case <-lctx.Done():
			return
		case msgs <- msg:
		}

	}, opt, nats.ManualAck())
	if err != nil {
		cancel()
		return fmt.Errorf("projector: failed to subscribe to subject %s: %w", subject, err)
	}
	num, err := sub.InitialConsumerPending()
	if err != nil {
		cancel()
		return fmt.Errorf("projector: failed to get max pending for subject %s: %w", subject, err)
	}
	_ = num
	if num <= 0 {
		cancel()
		return fmt.Errorf("projector: no events found for subject %s", subject)
	}

	defer close(msgs)
	defer sub.Unsubscribe()

	go func() {
		n := uint64(0)
		for {
			select {
			case <-ctx.Done():
				cancel()
				return
			case msg := <-msgs:
				if msg == nil {
					continue
				}
				n++

				var event = &gen.EventEnvelope{}
				if err := proto.Unmarshal(msg.Data, event); err != nil {
					_ = err
				}

				fn.ApplyEvent(event)
				msg.Ack()
				if n == num {
					cancel()
					return
				}
			}
		}
	}()
	<-lctx.Done()
	return nil
}

// ReplayAndSubscribe replays events for a given aggregate and aggregate ID,
// and subscribes to new events.
// It accepts ro.Options to configure the replay behavior.
// ro.WithAggregate ro.WithAggregateID - configure the aggregate and aggregate ID
// ro.WithSubject - use custom subject instead of default "events.aggregate.aggregateID.>"
// ro.WithStartSeq - start from event (if you have snapshot)
// ro.WtihParent - nests subjects
// ro.WithTimeout - timeout if no events for stream
func ReplayAndSubscribe[T EventApplier](ctx context.Context, agg T, opts ...ro.Options) <-chan T {
	cfg := &ro.Config{
		StartSeq: DeliverAll,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	subject := fmt.Sprintf("%s.%s.%s.>", EventsPrefix, cfg.Aggregate, cfg.AggregateID)
	if cfg.Subject != "" {
		subject = cfg.Subject
	}

	ch := make(chan T, 128)
	msgs := make(chan *nats.Msg, 128)
	js, _ := JetStream(ctx)
	sub, err := js.Subscribe(subject, func(msg *nats.Msg) {
		msgs <- msg
	}, nats.ManualAck(), nats.DeliverNew())
	if err != nil {
		log.Printf("ReplayAndSubscribe: Error subscribing to %s.%s: %v", cfg.Aggregate, cfg.AggregateID, err)
		ch <- agg
		close(ch)
		return ch
	}
	Replay(ctx, agg, opts...)
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
