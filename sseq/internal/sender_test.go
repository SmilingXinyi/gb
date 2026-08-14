package ss

import (
	"bytes"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type recordingWriter struct {
	mutex    sync.Mutex
	payloads [][]byte
	err      error
}

func (writer *recordingWriter) WritePayload(payload []byte) error {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	writer.payloads = append(writer.payloads, append([]byte(nil), payload...))
	return writer.err
}

func (writer *recordingWriter) Close() error {
	return nil
}

func (writer *recordingWriter) joined() string {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	return string(bytes.Join(writer.payloads, nil))
}

type identityEncoder struct{}

func (identityEncoder) Encode(event SpanEvent) ([]byte, error) {
	return []byte(event.Name), nil
}

type multilineEncoder struct{}

func (multilineEncoder) Encode(event SpanEvent) ([]byte, error) {
	return []byte(event.Name + "\n" + event.Name + "-event\n"), nil
}

func TestSenderFlushesOnBatchSize(t *testing.T) {
	writer := &recordingWriter{}
	sender := NewSender(BatchConfig{BatchSize: 2, FlushInterval: time.Hour}, identityEncoder{}, writer)
	if err := sender.Send(SpanEvent{Name: "one"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got := writer.joined(); got != "" {
		t.Fatalf("unexpected flush before batch size: %q", got)
	}
	if err := sender.Send(SpanEvent{Name: "two"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if err := sender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := writer.joined(); got != "one\ntwo\n" {
		t.Fatalf("payload = %q", got)
	}
}

func TestSenderCloseFlushesRemainder(t *testing.T) {
	writer := &recordingWriter{}
	sender := NewSender(BatchConfig{BatchSize: 10, FlushInterval: time.Hour}, identityEncoder{}, writer)
	if err := sender.Send(SpanEvent{Name: "leftover"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if err := sender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := writer.joined(); got != "leftover\n" {
		t.Fatalf("payload = %q", got)
	}
}

func TestSenderSendAfterClose(t *testing.T) {
	writer := &recordingWriter{}
	sender := NewSender(BatchConfig{BatchSize: 1, FlushInterval: time.Hour}, identityEncoder{}, writer)
	if err := sender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	err := sender.Send(SpanEvent{Name: "late"})
	if !errors.Is(err, ErrSenderClosed) {
		t.Fatalf("Send() error = %v, want ErrSenderClosed", err)
	}
}

func TestSenderDoesNotDoubleCountTrailingNewline(t *testing.T) {
	writer := &recordingWriter{}
	sender := NewSender(BatchConfig{BatchSize: 2, FlushInterval: time.Hour}, multilineEncoder{}, writer)
	if err := sender.Send(SpanEvent{Name: "span"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if err := sender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := writer.joined(); got != "span\nspan-event\n" {
		t.Fatalf("payload = %q", got)
	}
}

func TestSenderSerializesWrites(t *testing.T) {
	var inflight atomic.Int32
	var maxInflight atomic.Int32
	writer := &concurrencyWriter{inflight: &inflight, maxInflight: &maxInflight}
	sender := NewSender(BatchConfig{BatchSize: 1, FlushInterval: time.Hour}, identityEncoder{}, writer)

	var wait sync.WaitGroup
	for index := 0; index < 8; index++ {
		wait.Add(1)
		go func(name string) {
			defer wait.Done()
			if err := sender.Send(SpanEvent{Name: name}); err != nil {
				t.Errorf("Send() error = %v", err)
			}
		}("span")
	}
	wait.Wait()
	if err := sender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if maxInflight.Load() != 1 {
		t.Fatalf("max concurrent writes = %d, want 1", maxInflight.Load())
	}
}

func TestSenderQueueDoesNotBlockWhenCapacityAvailable(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	writer := &gateWriter{started: started, release: release}
	sender := NewSender(BatchConfig{
		BatchSize:      1,
		FlushInterval:  time.Hour,
		WriteQueueSize: 2,
	}, identityEncoder{}, writer)

	sendDone := make(chan error, 1)
	go func() {
		sendDone <- sender.Send(SpanEvent{Name: "queued"})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("writer did not start")
	}

	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Send blocked on slow writer while queue had capacity")
	}

	close(release)
	if err := sender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

type gateWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (writer *gateWriter) WritePayload(payload []byte) error {
	writer.once.Do(func() {
		close(writer.started)
	})
	<-writer.release
	return nil
}

func (writer *gateWriter) Close() error {
	return nil
}

func TestCountRecords(t *testing.T) {
	tests := []struct {
		payload string
		want    int
	}{
		{payload: "", want: 0},
		{payload: "a", want: 1},
		{payload: "a\n", want: 1},
		{payload: "a\nb", want: 2},
		{payload: "a\nb\n", want: 2},
		{payload: "\n", want: 0},
	}
	for _, test := range tests {
		if got := countRecords([]byte(test.payload)); got != test.want {
			t.Fatalf("countRecords(%q) = %d, want %d", test.payload, got, test.want)
		}
	}
}

type concurrencyWriter struct {
	inflight    *atomic.Int32
	maxInflight *atomic.Int32
}

func (writer *concurrencyWriter) WritePayload(payload []byte) error {
	current := writer.inflight.Add(1)
	defer writer.inflight.Add(-1)
	for {
		maxValue := writer.maxInflight.Load()
		if current <= maxValue {
			break
		}
		if writer.maxInflight.CompareAndSwap(maxValue, current) {
			break
		}
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (writer *concurrencyWriter) Close() error {
	return nil
}
