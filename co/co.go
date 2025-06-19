package co

type Options func(*Config)

type Config struct {
	Subject   string
	Aggregate string
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
