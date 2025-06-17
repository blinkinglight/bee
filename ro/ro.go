package ro

type Options func(*Config)

type Config struct {
	Subject     string
	Aggregate   string
	AggregateID string
	StartSeq    uint64
}

func WithSubject(subject string) Options {
	return func(cfg *Config) {
		cfg.Subject = subject
	}
}

func WithAggreate(aggregate string) Options {
	return func(cfg *Config) {
		cfg.Aggregate = aggregate
	}
}

func WithStartSeq(seq uint64) Options {
	return func(cfg *Config) {
		cfg.StartSeq = seq
	}
}

func WithAggregateID(id string) Options {
	return func(cfg *Config) {
		cfg.AggregateID = id
	}
}
