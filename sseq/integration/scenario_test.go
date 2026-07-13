package integration_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
)

const (
	integrationApplication = "sseq-integration"
	integrationSpanCount   = 5
)

const (
	rootSpanName = "Linear pipeline"

	spanStepA = "A"
	spanStepB = "B"
	spanStepC = "C"
	spanStepD = "D"
)

var integrationLinearSpanNames = []string{spanStepA, spanStepB, spanStepC, spanStepD}

type linearSpanStep struct {
	name     string
	duration time.Duration
}

var integrationLinearSteps = []linearSpanStep{
	{name: spanStepA, duration: 30 * time.Millisecond},
	{name: spanStepB, duration: 25 * time.Millisecond},
	{name: spanStepC, duration: 20 * time.Millisecond},
	{name: spanStepD, duration: 15 * time.Millisecond},
}

// integrationSpanRecord is the normalized span view used by integration assertions.
type integrationSpanRecord struct {
	Name         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	StartTime    time.Time
	Duration     time.Duration
}

// runIntegrationSpanScenario executes sequential sibling spans under one root:
//
//	Linear pipeline
//	├── A   (runs, completes)
//	├── B   (runs, completes)
//	├── C   (runs, completes)
//	└── D   (runs, completes)
func runIntegrationSpanScenario() (traceID string, err error) {
	requestContext, endRoot := sseq.Start(context.Background(), rootSpanName, "server")
	var ok bool
	traceID, _, ok = sseq.IDs(requestContext)
	if !ok {
		endRoot()
		return "", fmt.Errorf("missing trace id on root span")
	}

	for _, step := range integrationLinearSteps {
		stepErr := sseq.Trace(requestContext, step.name, "", func(context.Context) error {
			time.Sleep(step.duration)
			return nil
		})
		if stepErr != nil {
			endRoot()
			return traceID, stepErr
		}
	}

	endRoot()
	return traceID, nil
}

// verifyIntegrationSpanTree asserts sequential A → B → C → D execution as sibling spans.
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

	root, ok := spanByName[rootSpanName]
	if !ok {
		t.Fatalf("missing root span %q in %+v", rootSpanName, spanByName)
	}
	if root.ParentSpanID != "" {
		t.Fatalf("root span must not have parent_span_id, got %q", root.ParentSpanID)
	}

	for _, spanName := range integrationLinearSpanNames {
		span, found := spanByName[spanName]
		if !found {
			t.Fatalf("missing span %q", spanName)
		}
		if span.ParentSpanID != root.SpanID {
			t.Fatalf("span %q parent_span_id = %q, want %q (%q)", spanName, span.ParentSpanID, root.SpanID, rootSpanName)
		}
	}

	orderedSteps := make([]integrationSpanRecord, 0, len(integrationLinearSpanNames))
	for _, spanName := range integrationLinearSpanNames {
		orderedSteps = append(orderedSteps, spanByName[spanName])
	}

	for index := 1; index < len(orderedSteps); index++ {
		previous := orderedSteps[index-1]
		current := orderedSteps[index]
		assertStartsBefore(t, previous.Name, previous.StartTime, current.Name, current.StartTime)
		assertEndsBeforeStart(t, previous, current)
	}

	for _, step := range orderedSteps {
		assertNestedWithin(t, root, step)
	}

	if root.Duration > 0 {
		lastStep := orderedSteps[len(orderedSteps)-1]
		rootEnd := root.StartTime.Add(root.Duration)
		lastEnd := lastStep.StartTime.Add(lastStep.Duration)
		if lastEnd.After(rootEnd) {
			t.Fatalf("root span should end after step D: D ends %v, root ends %v", lastEnd, rootEnd)
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

func assertEndsBeforeStart(t *testing.T, earlier integrationSpanRecord, later integrationSpanRecord) {
	t.Helper()
	if earlier.StartTime.IsZero() || later.StartTime.IsZero() || earlier.Duration <= 0 {
		return
	}
	earlierEnd := earlier.StartTime.Add(earlier.Duration)
	if !earlierEnd.Before(later.StartTime) && !earlierEnd.Equal(later.StartTime) {
		t.Fatalf("expected %q to finish before %q starts: end=%v start=%v", earlier.Name, later.Name, earlierEnd, later.StartTime)
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

	if err := sseq.SetupSeqFile(filename, integrationApplication); err != nil {
		t.Fatalf("SetupSeqFile() error = %v", err)
	}
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
