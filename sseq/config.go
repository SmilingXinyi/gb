package sseq

import (
	"fmt"
	"os"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

const (
	// DefaultShutdownTimeout is how long Shutdown waits for in-flight spans.
	DefaultShutdownTimeout = 5 * time.Second
)

// Config controls batching, shutdown, and export error handling.
type Config struct {
	// BatchSize is how many encoded records trigger a flush.
	BatchSize int
	// FlushInterval is the maximum time between flushes.
	FlushInterval time.Duration
	// ShutdownTimeout is how long Shutdown waits for in-flight spans.
	ShutdownTimeout time.Duration
	// ErrorHandler receives export and shutdown errors. Nil uses stderr.
	ErrorHandler func(error)
}

// Option mutates Config during Setup.
type Option func(*Config)

// DefaultConfig returns the default exporter configuration.
func DefaultConfig() Config {
	return Config{
		BatchSize:       ss.DefaultBatchSize,
		FlushInterval:   ss.DefaultFlushInterval,
		ShutdownTimeout: DefaultShutdownTimeout,
		ErrorHandler:    defaultErrorHandler,
	}
}

// WithBatchSize sets how many encoded records trigger a flush.
func WithBatchSize(size int) Option {
	return func(config *Config) {
		config.BatchSize = size
	}
}

// WithFlushInterval sets the maximum time between flushes.
func WithFlushInterval(interval time.Duration) Option {
	return func(config *Config) {
		config.FlushInterval = interval
	}
}

// WithShutdownTimeout sets how long Shutdown waits for in-flight spans.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		config.ShutdownTimeout = timeout
	}
}

// WithErrorHandler sets the callback used for export and shutdown errors.
func WithErrorHandler(handler func(error)) Option {
	return func(config *Config) {
		config.ErrorHandler = handler
	}
}

// applyOptions applies Setup options on top of DefaultConfig.
func applyOptions(options ...Option) Config {
	config := DefaultConfig()
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&config)
	}
	if config.BatchSize <= 0 {
		config.BatchSize = ss.DefaultBatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = ss.DefaultFlushInterval
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = DefaultShutdownTimeout
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}
	return config
}

// defaultErrorHandler writes export errors to stderr.
func defaultErrorHandler(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "sseq: %v\n", err)
}
