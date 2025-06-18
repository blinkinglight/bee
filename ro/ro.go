package ro

import "time"

type Options func(*Config)

type Config struct {
	Subject     string
	Aggregate   string
	AggregateID string
	StartSeq    uint64
	Timeout     time.Duration
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

func WithTimeout(timeout time.Duration) Options {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}
