package bee

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/blinkinglight/bee/co"
	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const commandsStream = "COMMANDS"

type CommandHandler interface {
	Handle(m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error)
}

// type CommandHandlerFunc func(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error)

type CommandProcessor struct {
	js        nats.JetStreamContext
	nc        *nats.Conn
	aggregate string
	subject   string
	durable   string
	handler   CommandHandler
	cfg       *co.Config
}

// Command is the main entry point for processing commands.
// accepts co.Options to configure the command processor.
// co.WithSubject - use custom subject instead of default "cmds.aggregate"
// co.WithAggregate - use custom aggregate name
func Command(ctx context.Context, handler CommandHandler, opts ...co.Options) {
	cfg := &co.Config{}

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
	cp := &CommandProcessor{js: js, nc: nc, subject: subject, aggregate: cfg.Aggregate, durable: "default", handler: handler, cfg: cfg}
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

func (cp *CommandProcessor) init(ctx context.Context, cancel context.CancelFunc) error {

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
			var cmd gen.CommandEnvelope
			err := proto.Unmarshal(msg.Data, &cmd)
			if err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				msg.Ack()
				continue
			}

			errorNotificationSubject := fmt.Sprintf("notifications.%s.error", cmd.CorrelationId)
			successNotificationSubject := fmt.Sprintf("notifications.%s.success", cmd.CorrelationId)

			events, err := cp.handler.Handle(&cmd)
			if err != nil {
				log.Printf("Error handling command: %v", err)
				if cmd.CorrelationId != "" {
					cp.nc.Publish(errorNotificationSubject, []byte(`{"message":"`+err.Error()+`"}`))
				}
				msg.Ack()
				continue
			}

			for _, event := range events {
				eventSubject := fmt.Sprintf("events.%s.%s.%s", event.AggregateType, event.AggregateId, event.EventType)
				if len(event.Parents) > 0 {
					var parents []string
					for _, parent := range event.Parents {
						parents = append(parents, fmt.Sprintf("%s.%s", parent.AggregateType, parent.AggregateId))
					}
					eventSubject = fmt.Sprintf("events.%s.%s.%s.%s", strings.Join(parents, "."), event.AggregateType, event.AggregateId, event.EventType)
				}

				b, _ := proto.Marshal(event)
				if _, err := cp.js.Publish(eventSubject, b); err != nil {
					log.Printf("Error publishing event %v", err)
				}
			}

			if cmd.CorrelationId != "" {
				cp.nc.Publish(successNotificationSubject, []byte(`{"message":"`+cmd.AggregateId+`"}`))
			}
			_ = msg.Ack()
		}
	}
}
