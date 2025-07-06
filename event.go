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

/*
type ProcMgr struct {
	js        nats.JetStreamContext
	nc        *nats.Conn
	aggregate string
	subject   string
	durable   string
	handler   ManagerEventApplier
	cfg       *eo.Config
}

// Event is a process manager that listens for events and depending on situaion emits command events.
func Event(ctx context.Context, handler ManagerEventApplier, opts ...eo.Options) {
	cfg := &eo.Config{}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.Aggregate == "" {
		panic("Aggregate name is required for command processor")
	}

	subject := fmt.Sprintf(CommandsPrefix+".%s", cfg.Aggregate)
	if cfg.Subject != "" {
		subject = cfg.Subject
	}

	nc, _ := Nats(ctx)
	js, _ := JetStream(ctx)
	cp := &ProcMgr{js: js, nc: nc, subject: subject, aggregate: cfg.Aggregate, durable: "default", handler: handler, cfg: cfg}
	c, cancel := context.WithCancel(ctx)
	go func() {
		for {
			err := cp.init(c, cancel)
			if err != nil {
				log.Printf("Error initializing command processor: %v", err)
				continue
			}
		}
	}()
	<-c.Done()
}

func (cp *ProcMgr) init(ctx context.Context, cancel context.CancelFunc) error {

	_, _ = cp.js.AddStream(&nats.StreamConfig{
		Name:       commandsStream,
		Subjects:   []string{CommandsPrefix + ".>"},
		Retention:  nats.WorkQueuePolicy,
		Storage:    nats.FileStorage,
		Replicas:   1,
		Duplicates: 5 * time.Minute,
	})

	_, err := cp.js.AddConsumer(commandsStream, &nats.ConsumerConfig{
		Name:          cp.aggregate + "_" + cp.durable + "_cmd",
		Durable:       cp.aggregate + "_" + cp.durable + "_cmd",
		FilterSubject: cp.subject,
		AckPolicy:     nats.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	sub, err := cp.js.PullSubscribe(cp.subject, cp.aggregate+"_"+cp.durable+"_cmd", nats.BindStream(commandsStream), nats.ManualAck(), nats.AckExplicit(), nats.DeliverAll())

	if err != nil {
		log.Printf("Error subscribing to commands: %v", err)
		cancel()
	}
	defer sub.Unsubscribe()

	for {
		if ctx.Err() != nil {
			cancel()
			return nil
		}
		msg, _ := sub.Fetch(1, nats.MaxWait(60*time.Second))
		if len(msg) == 0 {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		for _, msg := range msg {
			var evt gen.EventEnvelope
			err := proto.Unmarshal(msg.Data, &evt)
			if err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				msg.Ack()
				continue
			}

			commands, err := cp.handler.Handle(&evt)
			if err != nil {
				msg.Ack()
				continue
			}

			for _, command := range commands {
				if command.Aggregate == "" {
					command.Aggregate = evt.AggregateType
				}
				if command.AggregateId == "" {
					command.AggregateId = evt.AggregateId
				}
				eventSubject := fmt.Sprintf(CommandsPrefix+".%s.%s.%s", command.Aggregate, command.AggregateId, command.CommandType)
				if len(command.Parents) > 0 {
					var parents []string
					for _, parent := range command.Parents {
						parents = append(parents, fmt.Sprintf("%s.%s", parent.AggregateType, parent.AggregateId))
					}
					eventSubject = fmt.Sprintf("events.%s.%s.%s.%s", strings.Join(parents, "."), command.Aggregate, command.AggregateId, command.CommandType)
				}

				b, _ := proto.Marshal(command)
				if _, err := cp.js.Publish(eventSubject, b); err != nil {
					log.Printf("Error publishing event %v", err)
				}
			}

			_ = msg.Ack()
		}
	}
}
*/
