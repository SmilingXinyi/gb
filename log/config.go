package log

// LogConfig defines the configuration for logging
type LogConfig struct {
	// Console defines the configuration for console output
	Console ConsoleConfig
	// File defines the configuration for file output
	File FileConfig
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
	}
}
