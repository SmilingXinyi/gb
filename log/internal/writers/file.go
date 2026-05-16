package writers

import (
	"io"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// FileWriterConfig defines the configuration for file logging
type FileWriterConfig struct {
	// Filename is the path to the log file
	Filename string
	// MaxSize is the maximum size of the log file in megabytes
	MaxSize int
	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int
	// MaxAge is the maximum number of days to retain old log files
	MaxAge int
	// Compress indicates whether old log files should be compressed
	Compress bool
}

// NewFileWriter creates a new file writer with rotation support
func NewFileWriter(config FileWriterConfig) io.Writer {
	if config.Filename == "" {
		return nil
	}
	return &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}
}
