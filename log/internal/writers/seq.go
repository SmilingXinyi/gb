package writers

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	defaultSeqBatchSize     = 50
	defaultSeqFlushInterval = 2 * time.Second
	seqCLEFContentType      = "application/vnd.serilog.clef"
)

// SeqWriterConfig defines settings for sending CLEF events to Seq.
type SeqWriterConfig struct {
	// Endpoint is the Seq CLEF ingestion URL, e.g. http://localhost:5342/ingest/clef.
	Endpoint string
	// APIKey is sent via the X-Seq-ApiKey header when non-empty.
	APIKey string
	// Application is added to every event as the Application property.
	Application string
	// BatchSize is the number of events to accumulate before flushing.
	BatchSize int
	// FlushInterval controls the maximum delay before buffered events are sent.
	FlushInterval time.Duration
	// HTTPClient overrides the default HTTP client, mainly for tests.
	HTTPClient *http.Client
}

// SeqWriter sends zerolog JSON events to Seq using the CLEF HTTP ingestion endpoint.
type SeqWriter struct {
	config     SeqWriterConfig
	httpClient *http.Client
	buffer     bytes.Buffer
	eventCount int
	mutex      sync.Mutex
	done       chan struct{}
	closed     bool
}

// NewSeqWriter creates an asynchronous Seq writer. Call Close to flush remaining events.
func NewSeqWriter(config SeqWriterConfig) *SeqWriter {
	if config.BatchSize <= 0 {
		config.BatchSize = defaultSeqBatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultSeqFlushInterval
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	writer := &SeqWriter{
		config:     config,
		httpClient: httpClient,
		done:       make(chan struct{}),
	}

	go writer.runFlushLoop()
	return writer
}

// Write converts a zerolog JSON payload into CLEF and queues it for delivery.
func (writer *SeqWriter) Write(payload []byte) (int, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return len(payload), nil
	}

	clefPayload, err := ConvertZerologJSONToCLEF(payload, writer.config.Application)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log seq writer: convert clef: %v\n", err)
		return len(payload), nil
	}

	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	if writer.closed {
		return len(payload), nil
	}

	writer.buffer.Write(clefPayload)
	writer.buffer.WriteByte('\n')
	writer.eventCount++

	if writer.eventCount >= writer.config.BatchSize {
		writer.flushLocked()
	}

	return len(payload), nil
}

// Close flushes buffered events and stops the background flush loop.
func (writer *SeqWriter) Close() error {
	writer.mutex.Lock()
	if writer.closed {
		writer.mutex.Unlock()
		return nil
	}
	writer.closed = true
	writer.flushLocked()
	writer.mutex.Unlock()

	close(writer.done)
	return nil
}

// runFlushLoop periodically flushes buffered events.
func (writer *SeqWriter) runFlushLoop() {
	ticker := time.NewTicker(writer.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			writer.mutex.Lock()
			writer.flushLocked()
			writer.mutex.Unlock()
		case <-writer.done:
			return
		}
	}
}

// flushLocked sends buffered events to Seq. The caller must hold writer.mutex.
func (writer *SeqWriter) flushLocked() {
	if writer.buffer.Len() == 0 {
		return
	}

	payload := append([]byte(nil), writer.buffer.Bytes()...)
	writer.buffer.Reset()
	writer.eventCount = 0

	go writer.sendPayload(payload)
}

// sendPayload posts a CLEF batch to Seq and ignores network failures.
func (writer *SeqWriter) sendPayload(payload []byte) {
	request, err := http.NewRequest(http.MethodPost, writer.config.Endpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "log seq writer: create request: %v\n", err)
		return
	}

	request.Header.Set("Content-Type", seqCLEFContentType)
	if writer.config.APIKey != "" {
		request.Header.Set("X-Seq-ApiKey", writer.config.APIKey)
	}

	response, err := writer.httpClient.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log seq writer: send request: %v\n", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "log seq writer: unexpected status %d\n", response.StatusCode)
	}
}
