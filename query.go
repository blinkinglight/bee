package bee

import (
	"context"
	"encoding/json"

	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/qo"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

func Query(ctx context.Context, fn Querier, opts ...qo.Options) error {
	cfg := &qo.Config{
		Aggregate: "*",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	subject := "query." + cfg.Aggregate + ".get"
	if cfg.Subject != "" {
		subject = cfg.Subject
	}

	nc, _ := Nats(ctx)

	_, _ = nc.QueueSubscribe(subject, cfg.Aggregate, func(msg *nats.Msg) {
		if msg == nil {
			return
		}

		query := &gen.QueryEnvelope{}
		if err := proto.Unmarshal(msg.Data, query); err != nil {
			msg.Respond([]byte(err.Error()))
			return
		}

		result, err := fn.Query(query)
		if err != nil {
			msg.Respond([]byte(err.Error()))
			return
		}
		response, err := json.Marshal(result)
		if err != nil {
			msg.Respond([]byte(err.Error()))
			return
		}
		msg.Respond(response)

	})

	<-ctx.Done()
	return nil
}
