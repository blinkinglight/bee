package po

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

// WithDurable sets the durable name for the subscription
func WithDurable(name string) Options {
	return func(cfg *Config) {
		cfg.DurableName = name
	}
}

// WithAggreate sets the aggregate type for the subscription
func WithAggreate(aggregate string) Options {
	return func(cfg *Config) {
		cfg.Aggregate = aggregate
	}
}

// WithAggrateID sets the aggregate ID for the subscription
func WithAggrateID(aggregateID string) Options {
	return func(cfg *Config) {
		cfg.AggregateID = aggregateID
	}
}

// WithPrefix sets a prefix for the durable name
func WithPrefix(prefix string) Options {
	return func(cfg *Config) {
		cfg.Prefix = prefix
	}
}
