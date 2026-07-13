package file

import (
	"fmt"
	"os"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultMaxSize    = 100
	defaultMaxBackups = 5
	defaultMaxAge     = 30
)

// Writer appends encoded span batches to a rotated file for Vector pickup.
type Writer struct {
	logger *lumberjack.Logger
}

// NewWriter creates a rotated file writer.
func NewWriter(filename string) (*Writer, error) {
	if filename == "" {
		return nil, fmt.Errorf("file writer requires filename")
	}
	return &Writer{
		logger: &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    defaultMaxSize,
			MaxBackups: defaultMaxBackups,
			MaxAge:     defaultMaxAge,
			Compress:   true,
		},
	}, nil
}

// WritePayload appends a batch to the rotated file.
func (writer *Writer) WritePayload(payload []byte) {
	if _, err := writer.logger.Write(payload); err != nil {
		fmt.Fprintf(os.Stderr, "sseq: write file: %v\n", err)
	}
}

// Close releases file resources.
func (writer *Writer) Close() error {
	return nil
}
