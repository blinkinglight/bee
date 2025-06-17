package bee

import (
	"context"
	"encoding/json"

	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

func Query(ctx context.Context, aggregate string, fn Querier) error {
	nc, _ := Nats(ctx)

	_, _ = nc.QueueSubscribe("query."+aggregate+".get", aggregate, func(msg *nats.Msg) {
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
