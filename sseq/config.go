package sseq

import (
	"net/http"
	"time"
)

// Provider selects the span delivery backend.
type Provider string

const (
	// ProviderHTTP delivers spans to a Seq ingestion endpoint.
	ProviderHTTP Provider = "http"
	// ProviderFile delivers spans to a rotated span file.
	ProviderFile Provider = "file"
	// ProviderAxiom delivers spans directly to an Axiom dataset.
	ProviderAxiom Provider = "axiom"
)

// FileFormat selects the on-disk span encoding for the file provider.
type FileFormat string

const (
	// FileFormatCLEF writes Seq-compatible CLEF JSON lines.
	FileFormatCLEF FileFormat = "clef"
	// FileFormatAxiom writes Axiom-compatible NDJSON for Vector file sources.
	FileFormatAxiom FileFormat = "axiom"
)

// Config defines span delivery settings.
type Config struct {
	// Provider selects the delivery backend. When empty, auto-detection prefers file,
	// then Axiom, then HTTP based on which backend fields are populated.
	Provider Provider
	// Application is added to every span as the Application property or service.name.
	Application string
	// BatchSize is the number of span events to accumulate before flushing.
	BatchSize int
	// FlushInterval controls the maximum delay before buffered spans are sent.
	FlushInterval time.Duration
	// HTTP configures the Seq HTTP provider.
	HTTP HTTPConfig
	// File configures the rotated file provider.
	File FileConfig
	// Axiom configures the direct Axiom ingest provider.
	Axiom AxiomConfig

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
	// Filename is the path to the span file.
	Filename string
	// MaxSize is the maximum size of each file in megabytes.
	MaxSize int
	// MaxBackups is the maximum number of old files to retain.
	MaxBackups int
	// MaxAge is the maximum number of days to retain old files.
	MaxAge int
	// Compress indicates whether old files should be compressed.
	Compress bool
	// Format selects the on-disk encoding. Defaults to CLEF.
	Format FileFormat
}

// AxiomConfig defines direct Axiom ingest settings.
type AxiomConfig struct {
	// Token is the Axiom API token with ingest permissions.
	Token string
	// Dataset is the Axiom dataset name that receives trace events.
	Dataset string
	// Domain is the Axiom API domain, e.g. api.axiom.co.
	Domain string
	// Endpoint optionally overrides the default ingest URL.
	Endpoint string
	// HTTPClient optionally overrides the default HTTP client.
	HTTPClient *http.Client
}

// DefaultConfig returns sensible defaults for the Seq HTTP provider.
func DefaultConfig() Config {
	return Config{
		Provider: ProviderHTTP,
		Endpoint: "http://localhost:5342/ingest/clef",
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
			Format:     FileFormatCLEF,
		},
		BatchSize:     20,
		FlushInterval: time.Second,
	}
}

// DefaultAxiomConfig returns sensible defaults for the Axiom provider.
func DefaultAxiomConfig() Config {
	return Config{
		Provider: ProviderAxiom,
		Axiom: AxiomConfig{
			Domain: "api.axiom.co",
		},
		BatchSize:     20,
		FlushInterval: time.Second,
	}
}

// DefaultAxiomVectorFileConfig returns file defaults for a Vector -> Axiom pipeline.
func DefaultAxiomVectorFileConfig() Config {
	return Config{
		Provider: ProviderFile,
		File: FileConfig{
			Filename:   "spans.ndjson",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
			Format:     FileFormatAxiom,
		},
		BatchSize:     20,
		FlushInterval: time.Second,
	}
}
