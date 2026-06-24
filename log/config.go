package log

import "time"

// LogConfig defines the configuration for logging
type LogConfig struct {
	// Console defines the configuration for console output
	Console ConsoleConfig
	// File defines the configuration for file output
	File FileConfig
	// Seq defines the configuration for Seq ingestion
	Seq SeqConfig
}

// ConsoleConfig defines the configuration for console output
type ConsoleConfig struct {
	// Enabled indicates whether console output is enabled
	Enabled bool
	// Level defines the log level (trace, debug, info, warn, error, fatal, panic)
	// If empty, it defaults to trace when Enabled is true, and info when false
	Level string
}

// FileConfig defines the configuration for file output
type FileConfig struct {
	// Filename is the path to the log file. If empty, file output is disabled
	Filename string
	// MaxSize is the maximum size of each log file in megabytes
	MaxSize int
	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int
	// MaxAge is the maximum number of days to retain old log files
	MaxAge int
	// Compress indicates whether old log files should be compressed
	Compress bool
}

// SeqConfig defines the configuration for Seq CLEF ingestion.
type SeqConfig struct {
	// Enabled turns on Seq output when true.
	Enabled bool
	// Endpoint is the Seq CLEF ingestion URL, e.g. http://localhost:5342/ingest/clef.
	Endpoint string
	// APIKey is sent via the X-Seq-ApiKey header when non-empty.
	APIKey string
	// Level defines the minimum log level sent to Seq.
	Level string
	// Application is added to every Seq event as the Application property.
	Application string
	// BatchSize is the number of events to accumulate before flushing.
	BatchSize int
	// FlushInterval controls the maximum delay before buffered events are sent.
	FlushInterval time.Duration
}

// DefaultConfig returns the default logging configuration
func DefaultConfig() LogConfig {
	return LogConfig{
		Console: ConsoleConfig{
			Enabled: true,
			Level:   "trace",
		},
		File: FileConfig{
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
		Seq: SeqConfig{
			Enabled:       false,
			Level:         "info",
			BatchSize:     50,
			FlushInterval: 2 * time.Second,
		},
	}
}
