package sseq_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
)

const (
	integrationApplication = "sseq-integration"
	integrationSpanCount   = 4
)

// Linear integration span names. The waterfall should read top-to-bottom as A → B → C → D.
const (
	spanStepA = "A"
	spanStepB = "B"
	spanStepC = "C"
	spanStepD = "D"
)

var integrationLinearSpanNames = []string{spanStepA, spanStepB, spanStepC, spanStepD}

// integrationSpanRecord is the normalized span view used by integration assertions.
type integrationSpanRecord struct {
	Name         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	StartTime    time.Time
	Duration     time.Duration
}

// runIntegrationSpanScenario executes a linear span chain where each step runs after the previous one:
//
//	A
//	└── B
//	    └── C
//	        └── D
func runIntegrationSpanScenario() (traceID string, err error) {
	requestContext, spanA := sseq.Start(context.Background(), spanStepA)
	traceID = spanA.TraceID()

	time.Sleep(30 * time.Millisecond)

	err = sseq.Do(requestContext, spanStepB, func(stepBContext context.Context) error {
		time.Sleep(25 * time.Millisecond)
		return sseq.Do(stepBContext, spanStepC, func(stepCContext context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return sseq.Do(stepCContext, spanStepD, func(context.Context) error {
				time.Sleep(15 * time.Millisecond)
				return nil
			})
		})
	})
	spanA.End()
	return traceID, err
}

// verifyIntegrationSpanTree asserts the linear A → B → C → D chain: parent links and sequential timing.
func verifyIntegrationSpanTree(t *testing.T, traceID string, spans []integrationSpanRecord) {
	t.Helper()

	if len(spans) < integrationSpanCount {
		t.Fatalf("expected at least %d spans, got %d: %+v", integrationSpanCount, len(spans), spans)
	}

	spanByName := make(map[string]integrationSpanRecord, len(spans))
	for _, span := range spans {
		if span.TraceID != "" && span.TraceID != traceID {
			t.Fatalf("span %q trace_id = %q, want %q", span.Name, span.TraceID, traceID)
		}
		spanByName[span.Name] = span
	}

	for index, spanName := range integrationLinearSpanNames {
		span, found := spanByName[spanName]
		if !found {
			t.Fatalf("missing span %q in %+v", spanName, spanByName)
		}

		if index == 0 {
			if span.ParentSpanID != "" {
				t.Fatalf("root span %q must not have parent_span_id, got %q", spanName, span.ParentSpanID)
			}
			continue
		}

		parentName := integrationLinearSpanNames[index-1]
		parent := spanByName[parentName]
		if span.ParentSpanID != parent.SpanID {
			t.Fatalf("span %q parent_span_id = %q, want %q (%q)", spanName, span.ParentSpanID, parent.SpanID, parentName)
		}
	}

	stepA := spanByName[spanStepA]
	stepB := spanByName[spanStepB]
	stepC := spanByName[spanStepC]
	stepD := spanByName[spanStepD]

	assertStartsBefore(t, stepA.Name, stepA.StartTime, stepB.Name, stepB.StartTime)
	assertStartsBefore(t, stepB.Name, stepB.StartTime, stepC.Name, stepC.StartTime)
	assertStartsBefore(t, stepC.Name, stepC.StartTime, stepD.Name, stepD.StartTime)

	assertNestedWithin(t, stepA, stepB)
	assertNestedWithin(t, stepB, stepC)
	assertNestedWithin(t, stepC, stepD)

	if stepA.Duration > 0 && stepD.Duration > 0 {
		chainEnd := stepD.StartTime.Add(stepD.Duration)
		rootEnd := stepA.StartTime.Add(stepA.Duration)
		if chainEnd.After(rootEnd) {
			t.Fatalf("linear chain should finish inside span A: D ends %v, A ends %v", chainEnd, rootEnd)
		}
	}
}

func assertStartsBefore(t *testing.T, earlierName string, earlierStart time.Time, laterName string, laterStart time.Time) {
	t.Helper()
	if earlierStart.IsZero() || laterStart.IsZero() {
		return
	}
	if !earlierStart.Before(laterStart) {
		t.Fatalf("expected %q to start before %q: %v vs %v", earlierName, laterName, earlierStart, laterStart)
	}
}

// assertNestedWithin verifies that child span runs entirely inside parent span time bounds.
func assertNestedWithin(t *testing.T, parent integrationSpanRecord, child integrationSpanRecord) {
	t.Helper()
	if parent.StartTime.IsZero() || child.StartTime.IsZero() || parent.Duration <= 0 || child.Duration <= 0 {
		return
	}
	parentEnd := parent.StartTime.Add(parent.Duration)
	childEnd := child.StartTime.Add(child.Duration)

	if child.StartTime.Before(parent.StartTime) {
		t.Fatalf("expected %q to start inside %q: child=%v parent=%v", child.Name, parent.Name, child.StartTime, parent.StartTime)
	}
	if childEnd.After(parentEnd) {
		t.Fatalf("expected %q to end inside %q: childEnd=%v parentEnd=%v", child.Name, parent.Name, childEnd, parentEnd)
	}
}

// TestIntegrationSpanScenarioClef runs the shared scenario against the file provider and
// verifies the resulting waterfall locally without external services.
func TestIntegrationSpanScenarioClef(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "integration-spans.clef")

	sseq.Setup(sseq.Config{
		Provider:      sseq.ProviderFile,
		Application:   integrationApplication,
		BatchSize:     1,
		FlushInterval: 50 * time.Millisecond,
		File: sseq.FileConfig{
			Filename: filename,
		},
	})
	t.Cleanup(sseq.Shutdown)

	traceID, err := runIntegrationSpanScenario()
	if err != nil {
		t.Fatalf("runIntegrationSpanScenario() error = %v", err)
	}
	if traceID == "" {
		t.Fatal("expected trace id from linear span chain")
	}
	sseq.Shutdown()

	spans, err := readClefSpanRecords(filename)
	if err != nil {
		t.Fatalf("readClefSpanRecords() error = %v", err)
	}
	if len(spans) != integrationSpanCount {
		t.Fatalf("expected exactly %d spans, got %d", integrationSpanCount, len(spans))
	}

	verifyIntegrationSpanTree(t, traceID, spans)
}

// readClefSpanRecords parses CLEF span events written by the file provider.
func readClefSpanRecords(filename string) ([]integrationSpanRecord, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var spans []integrationSpanRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var clefEvent map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &clefEvent); err != nil {
			return nil, err
		}

		startTime, err := parseClefTime(clefEvent["@st"])
		if err != nil {
			return nil, err
		}
		endTime, err := parseClefTime(clefEvent["@t"])
		if err != nil {
			return nil, err
		}

		spans = append(spans, integrationSpanRecord{
			Name:         stringField(clefEvent["@mt"]),
			TraceID:      stringField(clefEvent["@tr"]),
			SpanID:       stringField(clefEvent["@sp"]),
			ParentSpanID: stringField(clefEvent["@ps"]),
			StartTime:    startTime,
			Duration:     endTime.Sub(startTime),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return spans, nil
}

func parseClefTime(value any) (time.Time, error) {
	raw := stringField(value)
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, raw)
	}
	return parsed, err
}

func stringField(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}
