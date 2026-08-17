package application

import (
	"time"

	"github.com/google/uuid"
)

// Option is the shared functional option type for every service constructor.
type Option func(*serviceOptions)

// serviceOptions is the shared configuration for every application service.
type serviceOptions struct {
	idgen        func() string
	timeout      time.Duration
	tickInterval time.Duration
}

// defaultOptions returns the default service options.
func defaultOptions() serviceOptions {
	return serviceOptions{
		idgen:        uuid.NewString,
		timeout:      5 * time.Second,
		tickInterval: time.Minute,
	}
}

// WithIDGen overrides the default id generator.
func WithIDGen(fn func() string) Option {
	return func(o *serviceOptions) { o.idgen = fn }
}

// WithTimeout overrides the default I/O timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *serviceOptions) { o.timeout = d }
}

// WithTickInterval overrides the scheduler tick interval.
func WithTickInterval(d time.Duration) Option {
	return func(o *serviceOptions) { o.tickInterval = d }
}
