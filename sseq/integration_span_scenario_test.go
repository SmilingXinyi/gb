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
	integrationSpanCount   = 6
)

const (
	rootSpanName = "HTTP POST /api/orders"

	spanValidateAPIKey      = "1. Validate API key"
	spanFetchOrderDetails   = "2. Fetch order details"
	spanQueryOrdersTable    = "2.1 Query orders table"
	spanQueryOrderItems     = "2.2 Query order_items table"
	spanFormatJSONResponse  = "3. Format JSON response"
)

// integrationSpanRecord is the normalized span view used by integration assertions.
type integrationSpanRecord struct {
	Name         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	StartTime    time.Time
	Duration     time.Duration
}

// runIntegrationSpanScenario executes a sequential checkout-style span tree:
//
//	HTTP POST /api/orders
//	├── 1. Validate API key
//	├── 2. Fetch order details
//	│   ├── 2.1 Query orders table
//	│   └── 2.2 Query order_items table
//	└── 3. Format JSON response
func runIntegrationSpanScenario() (traceID string, err error) {
	requestContext, rootSpan := sseq.Start(context.Background(), rootSpanName)
	traceID = rootSpan.TraceID()

	if err := sseq.Do(requestContext, spanValidateAPIKey, func(ctx context.Context) error {
		time.Sleep(20 * time.Millisecond)
		return nil
	}); err != nil {
		rootSpan.End()
		return traceID, err
	}

	if err := sseq.Do(requestContext, spanFetchOrderDetails, func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond)
		if err := sseq.Do(ctx, spanQueryOrdersTable, func(ctx context.Context) error {
			time.Sleep(35 * time.Millisecond)
			return nil
		}); err != nil {
			return err
		}
		return sseq.Do(ctx, spanQueryOrderItems, func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})
	}); err != nil {
		rootSpan.End()
		return traceID, err
	}

	if err := sseq.Do(requestContext, spanFormatJSONResponse, func(ctx context.Context) error {
		time.Sleep(15 * time.Millisecond)
		return nil
	}); err != nil {
		rootSpan.End()
		return traceID, err
	}

	rootSpan.End()
	return traceID, nil
}

// verifyIntegrationSpanTree asserts hierarchy, ordering, and timing for the scenario tree.
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

	expectedParents := map[string]string{
		spanValidateAPIKey:     rootSpanName,
		spanFetchOrderDetails:  rootSpanName,
		spanQueryOrdersTable:   spanFetchOrderDetails,
		spanQueryOrderItems:    spanFetchOrderDetails,
		spanFormatJSONResponse: rootSpanName,
	}
	for spanName, parentName := range expectedParents {
		span, found := spanByName[spanName]
		if !found {
			t.Fatalf("missing span %q", spanName)
		}
		parent := spanByName[parentName]
		if span.ParentSpanID != parent.SpanID {
			t.Fatalf("span %q parent_span_id = %q, want %q (%q)", spanName, span.ParentSpanID, parent.SpanID, parentName)
		}
	}

	validate := spanByName[spanValidateAPIKey]
	fetch := spanByName[spanFetchOrderDetails]
	queryOrders := spanByName[spanQueryOrdersTable]
	queryItems := spanByName[spanQueryOrderItems]
	format := spanByName[spanFormatJSONResponse]

	assertStartsBefore(t, validate.Name, validate.StartTime, fetch.Name, fetch.StartTime)
	assertStartsBefore(t, fetch.Name, fetch.StartTime, format.Name, format.StartTime)
	assertStartsBefore(t, queryOrders.Name, queryOrders.StartTime, queryItems.Name, queryItems.StartTime)

	assertEndsBeforeStart(t, validate, fetch)
	assertEndsBeforeStart(t, validate, format)
	assertEndsBeforeStart(t, queryOrders, queryItems)
	assertEndsBeforeStart(t, queryItems, format)
	assertEndsBeforeStart(t, fetch, format)

	assertNestedWithin(t, fetch, queryOrders)
	assertNestedWithin(t, fetch, queryItems)

	if root.Duration > 0 && format.Duration > 0 {
		rootEnd := root.StartTime.Add(root.Duration)
		formatEnd := format.StartTime.Add(format.Duration)
		if formatEnd.After(rootEnd) {
			t.Fatalf("root span should end last: root ends %v, format ends %v", rootEnd, formatEnd)
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
