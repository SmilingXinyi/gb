package sender

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

const (
	defaultBatchSize     = 20
	defaultFlushInterval = time.Second
	defaultHTTPTimeout   = 10 * time.Second
)

// BatchConfig defines buffering settings shared by all providers.
type BatchConfig struct {
	BatchSize     int
	FlushInterval time.Duration
}

// HTTPBatchConfig defines HTTP provider settings with batching defaults.
type HTTPBatchConfig struct {
	Endpoint      string
	APIKey        string
	BatchSize     int
	FlushInterval time.Duration
	HTTPClient    *http.Client
}

// FileBatchConfig defines file provider settings with batching defaults.
type FileBatchConfig struct {
	File          FileConfig
	BatchSize     int
	FlushInterval time.Duration
}

// Sender batches span events and delivers them through a single PayloadWriter.
type Sender struct {
	config        BatchConfig
	writer        PayloadWriter
	buffer        bytes.Buffer
	eventCount    int
	mutex         sync.Mutex
	done          chan struct{}
	closed        bool
	flushLoopWait sync.WaitGroup
	postWait      sync.WaitGroup
}

// New creates an asynchronous span sender backed by the given payload writer.
func New(config BatchConfig, writer PayloadWriter) *Sender {
	if config.BatchSize <= 0 {
		config.BatchSize = defaultBatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultFlushInterval
	}

	sender := &Sender{
		config: config,
		writer: writer,
		done:   make(chan struct{}),
	}

	sender.flushLoopWait.Add(1)
	go sender.runFlushLoop()
	return sender
}

// NewHTTP creates a span sender that posts CLEF batches to Seq.
func NewHTTP(config HTTPBatchConfig) *Sender {
	batchConfig := BatchConfig{
		BatchSize:     config.BatchSize,
		FlushInterval: config.FlushInterval,
	}
	httpProvider := NewHTTPProvider(HTTPConfig{
		Endpoint:   config.Endpoint,
		APIKey:     config.APIKey,
		HTTPClient: config.HTTPClient,
	})
	return New(batchConfig, httpProvider)
}

// NewFile creates a span sender that writes CLEF batches to a rotated file.
func NewFile(config FileBatchConfig) (*Sender, error) {
	fileProvider, err := NewFileProvider(config.File)
	if err != nil {
		return nil, err
	}

	batchConfig := BatchConfig{
		BatchSize:     config.BatchSize,
		FlushInterval: config.FlushInterval,
	}
	return New(batchConfig, fileProvider), nil
}

// Send queues a span event for delivery.
func (sender *Sender) Send(event SpanEvent) error {
	payload, err := EncodeSpanEvent(event)
	if err != nil {
		return err
	}

	sender.mutex.Lock()
	defer sender.mutex.Unlock()

	if sender.closed {
		return nil
	}

	sender.buffer.Write(payload)
	sender.buffer.WriteByte('\n')
	sender.eventCount++

	if sender.eventCount >= sender.config.BatchSize {
		sender.flushLocked()
	}
	return nil
}

// Close flushes buffered events, stops the background flush loop, and waits for in-flight deliveries.
func (sender *Sender) Close() error {
	sender.mutex.Lock()
	if sender.closed {
		sender.mutex.Unlock()
		return nil
	}
	sender.closed = true
	sender.flushLocked()
	sender.mutex.Unlock()

	close(sender.done)
	sender.flushLoopWait.Wait()
	sender.postWait.Wait()
	if sender.writer != nil {
		return sender.writer.Close()
	}
	return nil
}

// runFlushLoop periodically flushes buffered span events.
func (sender *Sender) runFlushLoop() {
	defer sender.flushLoopWait.Done()

	ticker := time.NewTicker(sender.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sender.mutex.Lock()
			if !sender.closed {
				sender.flushLocked()
			}
			sender.mutex.Unlock()
		case <-sender.done:
			return
		}
	}
}

// flushLocked sends buffered events to the configured provider. The caller must hold sender.mutex.
func (sender *Sender) flushLocked() {
	if sender.buffer.Len() == 0 || sender.writer == nil {
		return
	}

	payload := append([]byte(nil), sender.buffer.Bytes()...)
	sender.buffer.Reset()
	sender.eventCount = 0

	sender.postWait.Add(1)
	go func() {
		defer sender.postWait.Done()
		sender.writer.WritePayload(payload)
	}()
}
