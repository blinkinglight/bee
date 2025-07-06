package eo

type Options func(*Config)

type Config struct {
	Subject     string
	Aggregate   string
	AggregateID string
	DurableName string
	Prefix      string
}

// WithSubject overrides default subject behavior
func WithSubject(subject string) Options {
	return func(cfg *Config) {
		cfg.Subject = subject
	}
}

// WithAggregateID sets the aggregate
func WithAggreate(aggregate string) Options {
	return func(cfg *Config) {
		cfg.Aggregate = aggregate
	}
}

// WithAggregateID sets the aggregate ID
func WithAggregateID(aggregateID string) Options {
	return func(cfg *Config) {
		cfg.AggregateID = aggregateID
	}
}

// WithDurableName sets the durable name for the subscription
func WithDurableName(durableName string) Options {
	return func(cfg *Config) {
		cfg.DurableName = durableName
	}
}

// WithPrefix sets the prefix for the subject
func WithPrefix(prefix string) Options {
	return func(cfg *Config) {
		cfg.Prefix = prefix
	}
}
