package ss

import (
	"bytes"
	"sync"
	"time"
)

// Sender batches encoded spans and flushes them through a Writer.
type Sender struct {
	config        BatchConfig
	encoder       Encoder
	writer        Writer
	buffer        bytes.Buffer
	eventCount    int
	mutex         sync.Mutex
	done          chan struct{}
	closed        bool
	flushLoopWait sync.WaitGroup
	postWait      sync.WaitGroup
}

// NewSender creates an asynchronous span sender.
func NewSender(config BatchConfig, encoder Encoder, writer Writer) *Sender {
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = DefaultFlushInterval
	}

	sender := &Sender{
		config:  config,
		encoder: encoder,
		writer:  writer,
		done:    make(chan struct{}),
	}
	sender.flushLoopWait.Add(1)
	go sender.runFlushLoop()
	return sender
}

// Send encodes and queues a span event for delivery.
func (sender *Sender) Send(event SpanEvent) error {
	payload, err := sender.encoder.Encode(event)
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
	sender.eventCount += countRecords(payload)
	if sender.eventCount >= sender.config.BatchSize {
		sender.flushLocked()
	}
	return nil
}

// Close flushes buffered events and releases the writer.
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

func countRecords(payload []byte) int {
	if len(payload) == 0 {
		return 0
	}
	count := 1
	for _, character := range payload {
		if character == '\n' {
			count++
		}
	}
	return count
}
