package sseq

import "time"

// Config defines Seq span ingestion settings.
type Config struct {
	// Endpoint is the Seq CLEF ingestion URL, e.g. http://localhost:5342/ingest/clef.
	Endpoint string
	// APIKey is sent via the X-Seq-ApiKey header when non-empty.
	APIKey string
	// Application is added to every span as the Application property.
	Application string
	// BatchSize is the number of span events to accumulate before flushing.
	BatchSize int
	// FlushInterval controls the maximum delay before buffered spans are sent.
	FlushInterval time.Duration
}

// DefaultConfig returns sensible defaults for local Seq.
func DefaultConfig() Config {
	return Config{
		Endpoint:      "http://localhost:5342/ingest/clef",
		BatchSize:     20,
		FlushInterval: time.Second,
	}
}
