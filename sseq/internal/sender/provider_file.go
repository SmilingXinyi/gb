package sender

import (
	"fmt"
	"os"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultFileMaxSize    = 100
	defaultFileMaxBackups = 5
	defaultFileMaxAge     = 30
)

// FileFormat selects the span encoding written to disk.
type FileFormat string

const (
	// FileFormatCLEF writes Seq-compatible CLEF JSON lines.
	FileFormatCLEF FileFormat = "clef"
	// FileFormatAxiom writes Axiom-compatible NDJSON for Vector file sources.
	FileFormatAxiom FileFormat = "axiom"
)

// FileConfig defines rotated file output settings for span events.
type FileConfig struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	// Format selects the on-disk encoding. Defaults to CLEF.
	Format FileFormat
}

// FileProvider writes CLEF batches to a rotated log file.
type FileProvider struct {
	logger *lumberjack.Logger
}

// NewFileProvider creates a rotated file payload writer.
func NewFileProvider(config FileConfig) (*FileProvider, error) {
	if config.Filename == "" {
		return nil, fmt.Errorf("file provider requires filename")
	}
	if config.MaxSize <= 0 {
		config.MaxSize = defaultFileMaxSize
	}
	if config.MaxBackups <= 0 {
		config.MaxBackups = defaultFileMaxBackups
	}
	if config.MaxAge <= 0 {
		config.MaxAge = defaultFileMaxAge
	}

	return &FileProvider{
		logger: &lumberjack.Logger{
			Filename:   config.Filename,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		},
	}, nil
}

// WritePayload appends a CLEF batch to the rotated span file.
func (provider *FileProvider) WritePayload(payload []byte) {
	if _, err := provider.logger.Write(payload); err != nil {
		fmt.Fprintf(os.Stderr, "sseq: write file: %v\n", err)
	}
}

// Close releases file provider resources.
func (provider *FileProvider) Close() error {
	return nil
}
