# bee - eventsourcing on nats.io

## experiment 

requires "https://buf.build/docs/" and `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`


examples:

`go run ./examples/subscribers`

and

`go run ./examples/publishers`


usage: 

```go
ctx = bee.WithNats(ctx, nc)
ctx = bee.WithJetStream(ctx, js)
go bee.NewCommandProcessor(ctx, NewService(), co.WithAggreate("users"))
go bee.Project(ctx, NewUserProjection(), po.WithAggreate("users"))
go bee.Query(ctx, NewUserProjection(), qo.WithAggreate("users"))
```

```go
agg := NewAggregate(m.AggregateId)
bee.Replay(ctx, agg, ro.WithAggreate(m.Aggregate), ro.WithAggregateID(m.AggregateId))
```

tiny example of live projectios: 

```go 
router.Get("/stream/{id}", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
    id := chi.URLParam(r, "id")
    sse := datastar.NewSSE(w, r)
    _ = sse

    ctx := bee.WithJetStream(r.Context(), js)
    ctx = bee.WithNats(ctx, nc)

    agg := &Aggregate{}
    updates := bee.ReplayAndSubscribe(ctx, agg, ro.WithAggreate(users.Aggregate), ro.WithAggregateID(id))
    for {
        select {
        case <-r.Context().Done():
            return
        case update := <-updates:
            sse.MergeFragmentTempl(partials.History(update.History))
        }
    }
})
```

and live projection aggrate: 

```go

type userModel struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

type Aggregate struct {
	History []string
}

func (a *Aggregate) ApplyEvent(event *gen.EventEnvelope) error {
	switch event.EventType {
	case "created":
		var user userModel
		if err := json.Unmarshal(event.Payload, &user); err != nil {
			return err
		}
		a.History = append(a.History, "User created: "+user.Name+" from "+user.Country)
	case "updated":
		var user userModel
		if err := json.Unmarshal(event.Payload, &user); err != nil {
			return err
		}
		a.History = append(a.History, "User updated: "+user.Name+" from "+user.Country)
	case "name_changed":
		var user userModel
		if err := json.Unmarshal(event.Payload, &user); err != nil {
			return err
		}
		a.History = append(a.History, "User name changed to: "+user.Name)
	default:
		return nil // Ignore other event types
	}
	return nil
}

```