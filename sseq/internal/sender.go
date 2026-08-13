package ss

import (
	"bytes"
	"errors"
	"sync"
	"time"
)

// ErrSenderClosed is returned when Send is called after Close.
var ErrSenderClosed = errors.New("sender is closed")

// Sender batches encoded spans and flushes them through a Writer.
type Sender struct {
	config        BatchConfig
	encoder       Encoder
	writer        Writer
	buffer        bytes.Buffer
	eventCount    int
	mutex         sync.Mutex
	writeMutex    sync.Mutex
	done          chan struct{}
	closed        bool
	flushLoopWait sync.WaitGroup
}

// NewSender creates a span sender that flushes by size or interval.
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
	if sender == nil || sender.encoder == nil {
		return nil
	}

	payload, err := sender.encoder.Encode(event)
	if err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}

	sender.mutex.Lock()
	if sender.closed {
		sender.mutex.Unlock()
		return ErrSenderClosed
	}

	sender.buffer.Write(payload)
	if payload[len(payload)-1] != '\n' {
		sender.buffer.WriteByte('\n')
	}
	sender.eventCount += countRecords(payload)

	var flushed []byte
	if sender.eventCount >= sender.config.BatchSize {
		flushed = sender.takeLocked()
	}
	sender.mutex.Unlock()

	if len(flushed) == 0 {
		return nil
	}
	return sender.write(flushed)
}

// Close flushes buffered events and releases the writer.
func (sender *Sender) Close() error {
	if sender == nil {
		return nil
	}

	sender.mutex.Lock()
	if sender.closed {
		sender.mutex.Unlock()
		return nil
	}
	sender.closed = true
	payload := sender.takeLocked()
	sender.mutex.Unlock()

	close(sender.done)
	sender.flushLoopWait.Wait()

	var closeErr error
	if len(payload) > 0 {
		closeErr = sender.write(payload)
	}

	sender.writeMutex.Lock()
	sender.writeMutex.Unlock()

	if sender.writer != nil {
		if err := sender.writer.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

// runFlushLoop periodically flushes buffered spans until Close.
func (sender *Sender) runFlushLoop() {
	defer sender.flushLoopWait.Done()
	ticker := time.NewTicker(sender.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sender.flush()
		case <-sender.done:
			return
		}
	}
}

// flush writes the current buffer if the sender is still open.
func (sender *Sender) flush() {
	sender.mutex.Lock()
	if sender.closed {
		sender.mutex.Unlock()
		return
	}
	payload := sender.takeLocked()
	sender.mutex.Unlock()

	if err := sender.write(payload); err != nil {
		sender.reportError(err)
	}
}

// takeLocked copies and resets the buffer. The caller must hold mutex.
func (sender *Sender) takeLocked() []byte {
	if sender.buffer.Len() == 0 {
		return nil
	}
	payload := append([]byte(nil), sender.buffer.Bytes()...)
	sender.buffer.Reset()
	sender.eventCount = 0
	return payload
}

// write delivers a payload through the writer, serializing concurrent flushes.
func (sender *Sender) write(payload []byte) error {
	if len(payload) == 0 || sender.writer == nil {
		return nil
	}

	sender.writeMutex.Lock()
	defer sender.writeMutex.Unlock()
	return sender.writer.WritePayload(payload)
}

// reportError invokes the optional export error callback.
func (sender *Sender) reportError(err error) {
	if err == nil || sender.config.OnError == nil {
		return
	}
	sender.config.OnError(err)
}

// countRecords counts NDJSON records, ignoring a trailing newline.
func countRecords(payload []byte) int {
	trimmed := bytes.TrimRight(payload, "\n")
	if len(trimmed) == 0 {
		return 0
	}
	return bytes.Count(trimmed, []byte{'\n'}) + 1
}
