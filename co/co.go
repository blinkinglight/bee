package co

type Options func(*Config)

type Config struct {
	Subject   string
	Aggregate string
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
