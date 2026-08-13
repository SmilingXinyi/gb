package file

import (
	"fmt"
	"sync"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultMaxSize    = 100
	defaultMaxBackups = 5
	defaultMaxAge     = 30
)

// Writer appends encoded span batches to a rotated file for Vector pickup.
type Writer struct {
	mutex  sync.Mutex
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
func (writer *Writer) WritePayload(payload []byte) error {
	if writer == nil {
		return fmt.Errorf("file writer is nil")
	}

	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	if writer.logger == nil {
		return fmt.Errorf("file writer is closed")
	}
	if _, err := writer.logger.Write(payload); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// Close flushes and closes the rotated file.
func (writer *Writer) Close() error {
	if writer == nil {
		return nil
	}

	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	if writer.logger == nil {
		return nil
	}
	err := writer.logger.Close()
	writer.logger = nil
	return err
}
