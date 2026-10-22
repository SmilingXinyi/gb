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

func TestGeneratedIDsAreW3CHex(t *testing.T) {
	tracer := NewTracer("demo", nil, time.Second, nil)
	ctx, span := tracer.Start(context.Background(), "root", "server")
	if !isHexID(span.traceID, 32) || !isHexID(span.spanID, 16) {
		t.Fatalf("root ids = (%s, %s)", span.traceID, span.spanID)
	}
	if span.parentID != "" {
		t.Fatalf("root parent = %q", span.parentID)
	}
	_, child := tracer.Start(ctx, "child", "")
	if !isHexID(child.spanID, 16) || child.traceID != span.traceID || child.parentID != span.spanID {
		t.Fatalf("child ids = (%s, %s, parent %s)", child.traceID, child.spanID, child.parentID)
	}
}

func TestResumeRejectsInvalidIDs(t *testing.T) {
	ctx := Resume(context.Background(), "not-a-trace", "bbbbbbbbbbbbbbbb", "01")
	if _, _, ok := IDsFromContext(ctx); ok {
		t.Fatal("expected invalid trace id to be ignored")
	}

	ctx = Resume(context.Background(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "short", "01")
	if _, _, ok := IDsFromContext(ctx); ok {
		t.Fatal("expected invalid parent span id to be ignored")
	}

	ctx = Resume(context.Background(), "00000000000000000000000000000000", "bbbbbbbbbbbbbbbb", "01")
	if _, _, ok := IDsFromContext(ctx); ok {
		t.Fatal("expected all-zero trace id to be ignored")
	}
}

func TestResumeKeepsFlagsAndNormalizesIDs(t *testing.T) {
	ctx := Resume(context.Background(), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "BBBBBBBBBBBBBBBB", "00")
	traceID, spanID, ok := IDsFromContext(ctx)
	if !ok {
		t.Fatal("expected resumed trace")
	}
	if traceID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || spanID != "bbbbbbbbbbbbbbbb" {
		t.Fatalf("ids = (%s, %s)", traceID, spanID)
	}
	if FlagsFromContext(ctx) != "00" {
		t.Fatalf("flags = %q", FlagsFromContext(ctx))
	}

	tracer := NewTracer("test", nil, time.Second, nil)
	childCtx, _ := tracer.Start(ctx, "child", "")
	if FlagsFromContext(childCtx) != "00" {
		t.Fatalf("child flags = %q", FlagsFromContext(childCtx))
	}
}

type identityEncoder struct{}

func (identityEncoder) Encode(event ss.SpanEvent) ([]byte, error) {
	return []byte(event.Name), nil
}
