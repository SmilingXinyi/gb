package sender

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	defaultBatchSize     = 20
	defaultFlushInterval = time.Second
	seqCLEFContentType   = "application/vnd.serilog.clef"
)

// Config defines HTTP delivery settings for span events.
type Config struct {
	Endpoint      string
	APIKey        string
	BatchSize     int
	FlushInterval time.Duration
	HTTPClient    *http.Client
}

// Sender batches and posts CLEF span events to Seq.
type Sender struct {
	config        Config
	httpClient    *http.Client
	buffer        bytes.Buffer
	eventCount    int
	mutex         sync.Mutex
	done          chan struct{}
	closed        bool
	flushLoopWait sync.WaitGroup
	postWait      sync.WaitGroup
}

// New creates an asynchronous Seq span sender.
func New(config Config) *Sender {
	if config.BatchSize <= 0 {
		config.BatchSize = defaultBatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = defaultFlushInterval
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	sender := &Sender{
		config:     config,
		httpClient: httpClient,
		done:       make(chan struct{}),
	}

	sender.flushLoopWait.Add(1)
	go sender.runFlushLoop()
	return sender
}

// Send queues a span event for delivery to Seq.
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

// flushLocked sends buffered events to Seq. The caller must hold sender.mutex.
func (sender *Sender) flushLocked() {
	if sender.buffer.Len() == 0 {
		return
	}

	payload := append([]byte(nil), sender.buffer.Bytes()...)
	sender.buffer.Reset()
	sender.eventCount = 0

	sender.postWait.Add(1)
	go func() {
		defer sender.postWait.Done()
		sender.postPayload(payload)
	}()
}

// postPayload delivers a CLEF batch to Seq and logs transport failures to stderr.
func (sender *Sender) postPayload(payload []byte) {
	request, err := http.NewRequest(http.MethodPost, sender.config.Endpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: create request: %v\n", err)
		return
	}

	request.Header.Set("Content-Type", seqCLEFContentType)
	if sender.config.APIKey != "" {
		request.Header.Set("X-Seq-ApiKey", sender.config.APIKey)
	}

	response, err := sender.httpClient.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: send request: %v\n", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "sseq: unexpected status %d\n", response.StatusCode)
	}
}
