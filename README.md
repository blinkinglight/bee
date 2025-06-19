# bee - eventsourcing on nats.io

## experiment 

examples:

`go run ./examples/subscribers`

and

`go run ./examples/publishers`

also 

`go run ./examples/query`


## Table of Contents

  - [Getting Started](#getting-started)
    - [Installing](#installing)
	- [Interfaces](#interfaces)
	- [Functions](#functions)
	- [Options](#options)
  - [Usage](#usage)
  - [Example](#example)
  - [Developing](#developing)
## Getting Started

### Installing 

To start using `bee`, install Go and run `go get`:
```sh
go get github.com/blinkinglight/bee
```

This will retrieve the library and update your `go.mod` and `go.sum` files.

### Interfaces 

```go

// Command handler
type CommandHandler interface {
	Handle(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error)
}

// Projection handler
type EventApplier interface {
	ApplyEvent(event *gen.EventEnvelope) error
}

// Query handler
type Querier interface {
	Query(query *gen.QueryEnvelope) (interface{}, error)
}

// Replay handler
type ReplayHandler interface {
	ApplyEvent(m *gen.EventEnvelope) error
}
```


### Functions

```go
func Command(ctx context.Context, handler CommandHandler, opts ...co.Options)
func Project(ctx context.Context, fn EventApplier, opts ...po.Options) error 
func Query(ctx context.Context, fn Querier, opts ...qo.Options) error 	
func Replay(ctx context.Context, fn ReplayHandler, opts ...ro.Options)
```

### Options

Command options
```go
func WithSubject(subject string) Options
func WithAggreate(aggregate string) Options
```

Projection options
```go
func WithSubject(subject string) Options
func WithDurable(name string) Options
func WithAggreate(aggregate string) Options
func WithAggrateID(aggregateID string) Options 
func WithPrefix(prefix string) Options
```

Query options
```go
func WithSubject(subject string) Options 
func WithAggreate(aggregate string) Options
```

Replay options
```go
func WithEventType(eventType string) Options
func WithParent(aggreate, id string) Options 
func WithSubject(subject string) Options
func WithAggreate(aggregate string) Options
func WithStartSeq(seq uint64) Options
func WithAggregateID(id string) Options
func WithTimeout(timeout time.Duration) Options
```



## Usage: 

```go
ctx = bee.WithNats(ctx, nc)
ctx = bee.WithJetStream(ctx, js)
go bee.Command(ctx, NewService(), co.WithAggreate("users"))
go bee.Project(ctx, NewUserProjection(), po.WithAggreate("users"))
go bee.Query(ctx, NewUserProjection(), qo.WithAggreate("users"))
```

```go
agg := NewAggregate(m.AggregateId)
bee.Replay(ctx, agg, ro.WithAggreate(m.Aggregate), ro.WithAggregateID(m.AggregateId))
```

## Example

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
type Aggregate struct {
	History []string
}

func (a *Aggregate) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	switch event := event.(type) {
	case *users.UserCreated:
		a.History = append(a.History, "User created: "+event.Name+" from "+event.Country)
	case *users.UserUpdated:
		a.History = append(a.History, "User updated: "+event.Name+" from "+event.Country)
	case *users.UserNameChanged:
		a.History = append(a.History, "User name changed to: "+event.Name)
	default:
		log.Printf("unknown event type: %T", event)
		return nil // Ignore other event types
	}
	return nil
}
```

## Developing

to work with this package you need 2 apps:

`https://buf.build/docs/` and `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`
