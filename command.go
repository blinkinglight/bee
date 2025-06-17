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

const commandsSubject = "cmds.>"
const commandsStream = "COMMANDS"

type CommandHandler interface {
	Handle(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error)
}

// type CommandHandlerFunc func(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error)

type CommandProcessor struct {
	js      nats.JetStreamContext
	nc      *nats.Conn
	subject string
	durable string
	handler CommandHandler
}

func NewCommandProcessor(ctx context.Context, subject string, handler CommandHandler) {
	nc, _ := Nats(ctx)
	js, _ := JetStream(ctx)
	cp := &CommandProcessor{js: js, nc: nc, subject: subject, durable: "default", handler: handler}
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
		Subjects:   []string{commandsSubject},
		Retention:  nats.WorkQueuePolicy,
		Storage:    nats.FileStorage,
		Replicas:   1,
		Duplicates: 5 * time.Minute,
	})

	_, err := cp.js.AddConsumer(commandsStream, &nats.ConsumerConfig{
		Name:          cp.subject + "_" + cp.durable + "_cmd",
		Durable:       cp.subject + "_" + cp.durable + "_cmd",
		FilterSubject: "cmds." + cp.subject,
		AckPolicy:     nats.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	sub, err := cp.js.PullSubscribe("cmds."+cp.subject, cp.subject+"_"+cp.durable+"_cmd", nats.BindStream(commandsStream), nats.ManualAck(), nats.DeliverAll())

	if err != nil {
		log.Printf("Error subscribing to commands: %v", err)
		cancel()
	}
	defer sub.Unsubscribe()

	for {
		msg, _ := sub.Fetch(1, nats.MaxWait(5*time.Second))
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

			events, err := cp.handler.Handle(ctx, &cmd)
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
	<-ctx.Done()
	return nil
}
