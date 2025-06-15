package bee

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
)

type QueryOne struct {
	ID string `json:"id"`
}

type QueryMany struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
}

func Query(ctx context.Context, nc *nats.Conn, aggregate string, fn Querier) error {
	_, _ = nc.Subscribe("query."+aggregate+".get", func(msg *nats.Msg) {
		if msg == nil {
			return
		}

		query := &QueryOne{}
		if err := json.Unmarshal(msg.Data, query); err != nil {
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
	_, _ = nc.Subscribe("query."+aggregate+".list", func(msg *nats.Msg) {
		if msg == nil {
			return
		}

		query := &QueryMany{}
		if err := json.Unmarshal(msg.Data, query); err != nil {
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
