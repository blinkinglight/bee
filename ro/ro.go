package ro

import "time"

type Options func(*Config)

type Config struct {
	Subject     string
	Aggregate   string
	AggregateID string
	StartSeq    uint64
	Timeout     time.Duration
	EventType   string
	Parents     []string
}

// WithEventType sets the EventType
func WithEventType(eventType string) Options {
	return func(cfg *Config) {
		cfg.EventType = eventType
	}
}

// WithParent appends a parent aggregate and ID to the Parents
// modifies subject into nested structure
// e.g. parent1.id1.parent2.id2.parent3.id3.aggregate.id
func WithParent(aggreate, id string) Options {
	return func(cfg *Config) {
		cfg.Parents = append(cfg.Parents, aggreate+"."+id)
	}
}

// WithSubject sets the Subject
func WithSubject(subject string) Options {
	return func(cfg *Config) {
		cfg.Subject = subject
	}
}

// WithAggreate sets the Aggregate
func WithAggreate(aggregate string) Options {
	return func(cfg *Config) {
		cfg.Aggregate = aggregate
	}
}

// WithStartSeq sets the StartSeq
func WithStartSeq(seq uint64) Options {
	return func(cfg *Config) {
		cfg.StartSeq = seq
	}
}

// WithAggregateID sets the AggregateID
func WithAggregateID(id string) Options {
	return func(cfg *Config) {
		cfg.AggregateID = id
	}
}

// WithTimeout sets the Timeout
func WithTimeout(timeout time.Duration) Options {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}
