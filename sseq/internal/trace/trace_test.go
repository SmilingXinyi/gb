package trace

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

type blockingWriter struct {
	payloads [][]byte
}

func (writer *blockingWriter) WritePayload(payload []byte) error {
	writer.payloads = append(writer.payloads, append([]byte(nil), payload...))
	return nil
}

func (writer *blockingWriter) Close() error {
	return nil
}

func TestCloseWaitsForInFlightSpan(t *testing.T) {
	writer := &blockingWriter{}
	sender := ss.NewSender(ss.BatchConfig{BatchSize: 1, FlushInterval: time.Hour}, identityEncoder{}, writer)
	tracer := NewTracer("test", sender, time.Second, nil)

	started := make(chan struct{})
	release := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		_ = tracer.Trace(context.Background(), "slow", "", func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	closed := make(chan struct{})
	go func() {
		if err := tracer.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close returned before in-flight span ended")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after span ended")
	}
	wait.Wait()

	if len(writer.payloads) == 0 {
		t.Fatal("expected flushed span after Close")
	}
}

type identityEncoder struct{}

func (identityEncoder) Encode(event ss.SpanEvent) ([]byte, error) {
	return []byte(event.Name), nil
}
