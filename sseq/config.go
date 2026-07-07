package sseq

import "time"

// Provider selects the span delivery backend.
type Provider string

const (
	// ProviderHTTP delivers spans to a Seq ingestion endpoint.
	ProviderHTTP Provider = "http"
	// ProviderFile delivers spans to a rotated CLEF file.
	ProviderFile Provider = "file"
)

// Config defines span delivery settings.
type Config struct {
	// Provider selects the delivery backend. When empty, HTTP is used if Endpoint is set,
	// otherwise file is used when File.Filename is set.
	Provider Provider
	// Application is added to every span as the Application property.
	Application string
	// BatchSize is the number of span events to accumulate before flushing.
	BatchSize int
	// FlushInterval controls the maximum delay before buffered spans are sent.
	FlushInterval time.Duration
	// HTTP configures the Seq HTTP provider.
	HTTP HTTPConfig
	// File configures the rotated file provider.
	File FileConfig

	// Endpoint is a shorthand for HTTP.Endpoint kept for backward compatibility.
	Endpoint string
	// APIKey is a shorthand for HTTP.APIKey kept for backward compatibility.
	APIKey string
}

// HTTPConfig defines Seq HTTP ingestion settings.
type HTTPConfig struct {
	// Endpoint is the Seq CLEF ingestion URL, e.g. http://localhost:5342/ingest/clef.
	Endpoint string
	// APIKey is sent via the X-Seq-ApiKey header when non-empty.
	APIKey string
}

// FileConfig defines rotated file output settings for span events.
type FileConfig struct {
	// Filename is the path to the span CLEF file.
	Filename string
	// MaxSize is the maximum size of each file in megabytes.
	MaxSize int
	// MaxBackups is the maximum number of old files to retain.
	MaxBackups int
	// MaxAge is the maximum number of days to retain old files.
	MaxAge int
	// Compress indicates whether old files should be compressed.
	Compress bool
}

// DefaultConfig returns sensible defaults for the Seq HTTP provider.
func DefaultConfig() Config {
	return Config{
		Provider:      ProviderHTTP,
		Endpoint:      "http://localhost:5342/ingest/clef",
		HTTP: HTTPConfig{
			Endpoint: "http://localhost:5342/ingest/clef",
		},
		BatchSize:     20,
		FlushInterval: time.Second,
	}
}

// DefaultFileConfig returns sensible defaults for the file provider.
func DefaultFileConfig() Config {
	return Config{
		Provider: ProviderFile,
		File: FileConfig{
			Filename:   "spans.clef",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
		BatchSize:     20,
		FlushInterval: time.Second,
	}
}
