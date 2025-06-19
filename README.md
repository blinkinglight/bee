# bee - eventsourcing on nats.io

Bee is a minimal Go library for implementing CQRS & Event-Sourcing using NATS JetStream as the transport and persistence layer. It offers clear abstractions with minimal dependencies and overhead.

## Why Bee?
- Minimal Infrastructure: Leverages your existing NATS JetStream.
- Type-Safe: Commands and events are protobuf-based.
- Fast Replay: Efficiently rebuild aggregate states from event streams.
- Small Footprint: Less than 1500 lines of simple, maintainable Go.

## Table of Contents

  - [Getting Started](#getting-started)
    - [Installing](#installing)
	- [Settings](#settings)
	- [Interfaces](#interfaces)
	- [Functions](#functions)
	- [Options](#options)
  - [Use cases](#typical-use-cases)
  - [Usage](#usage)
  - [Example](#example)
	- [Prebuild examples](#prebuild-examples)
  - [Roadmap](#roadmap)
  - [Developing](#developing)
  - [License](#license)
## Getting Started

### Installing 

To start using `bee`, install Go and run `go get`:
```sh
go get github.com/blinkinglight/bee
```

This will retrieve the library and update your `go.mod` and `go.sum` files.

### Settings

```go
bee.EventsPrefix = "events"
bee.CommandsPrefix = "cmds"
bee.QueryPrefix = "query"
```

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
func ReplayAndSubscribe[T EventApplier](ctx context.Context, agg T, opts ...ro.Options) <-chan T {
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


## Typical Use Cases

Bee is great for small, event-centric scenarios:

- Task-Oriented Microservices: Independent scaling of read/write sides.
- Audit Trails & Ledgers: Immutable event history for compliance.
- Sagas & Workflows: Event-driven state transitions.
- Edge/IoT Deployments: Compact deployment on resource-limited devices.
- Real-Time Game States: Fast catch-up of player states.
- SaaS On-Prem Plugins: Easy local deployment without infrastructure complexity.
- Ad-Hoc Analytics: Quickly spin up event projections.

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

### Prebuild examples

examples:

`go run ./examples/subscribers`

and

`go run ./examples/publishers`

also 

`go run ./examples/query`


## Developing

to work with this package you need 2 apps:

`https://buf.build/docs/` and `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`


## Roadmap

| Version | Planned Features                                     |
|---------|------------------------------------------------------|
| v0.3    | Snapshots						                     |
| v1.0    | Stable API, full pkg.go.dev docs                     |

## Development & Contribution

Pull requests are welcome!

## License

Apache-2.0 © 2025 BlinkLight