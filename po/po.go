package po

type Options func(*Config)

type Config struct {
	Subject     string
	Aggreate    string
	AggreateID  string
	DurableName string
	Prefix      string
}

func WithSubject(subject string) Options {
	return func(cfg *Config) {
		cfg.Subject = subject
	}
}

func WithDurable(name string) Options {
	return func(cfg *Config) {
		cfg.DurableName = name
	}
}

func WithAggreate(aggregate string) Options {
	return func(cfg *Config) {
		cfg.Aggreate = aggregate
	}
}

func WithAggrateID(aggregateID string) Options {
	return func(cfg *Config) {
		cfg.AggreateID = aggregateID
	}
}

func WithPrefix(prefix string) Options {
	return func(cfg *Config) {
		cfg.Prefix = prefix
	}
}
