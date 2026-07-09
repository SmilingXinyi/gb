package sseq_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
)

const (
	axiomQueryEndpoint = "https://api.axiom.co/v1/datasets/_apl?format=legacy"
	axiomPollAttempts  = 12
	axiomPollInterval  = 500 * time.Millisecond
)

func TestIntegrationSpanTreeWithAxiom(t *testing.T) {
	if os.Getenv("SSEQ_SKIP_INTEGRATION") == "1" {
		t.Skip("integration disabled by SSEQ_SKIP_INTEGRATION=1")
	}

	token := os.Getenv("AXIOM_TOKEN")
	dataset := os.Getenv("AXIOM_DATASET")
	if token == "" || dataset == "" {
		t.Skip("AXIOM_TOKEN and AXIOM_DATASET are required for Axiom integration test")
	}

	ingestTracker := newAxiomIngestTracker(http.DefaultTransport)
	httpClient := &http.Client{
		Transport: ingestTracker,
		Timeout:   30 * time.Second,
	}

	if err := sseq.Setup(sseq.Config{
		Provider:      sseq.ProviderAxiom,
		Application:   integrationApplication,
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		Axiom: sseq.AxiomConfig{
			Token:      token,
			Dataset:    dataset,
			Domain:     "api.axiom.co",
			HTTPClient: httpClient,
		},
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(sseq.Shutdown)

	traceID, err := runIntegrationSpanScenario()
	if err != nil {
		t.Fatalf("runIntegrationSpanScenario() error = %v", err)
	}
	if traceID == "" {
		t.Fatal("expected trace id from root span")
	}

	sseq.Shutdown()

	spans, queryErr := queryAxiomSpans(token, dataset, traceID)
	if queryErr == nil {
		verifyIntegrationSpanTree(t, traceID, spans)
		return
	}

	if !strings.Contains(queryErr.Error(), "token may lack query permission") {
		t.Fatalf("query axiom spans: %v", queryErr)
	}

	ingestedEvents, ingestRequests := ingestTracker.stats()
	if ingestRequests < integrationSpanCount {
		t.Fatalf("expected at least %d ingest requests, got %d", integrationSpanCount, ingestRequests)
	}
	if ingestedEvents < integrationSpanCount {
		t.Fatalf("expected at least %d ingested events, got %d (query unavailable: %v)", integrationSpanCount, ingestedEvents, queryErr)
	}

	t.Logf("query verification skipped (%v); ingest verified %d events across %d requests", queryErr, ingestedEvents, ingestRequests)
}

type axiomIngestTracker struct {
	base           http.RoundTripper
	requestCount   atomic.Int32
	ingestedEvents atomic.Int32
}

func newAxiomIngestTracker(base http.RoundTripper) *axiomIngestTracker {
	if base == nil {
		base = http.DefaultTransport
	}
	return &axiomIngestTracker{base: base}
}

func (tracker *axiomIngestTracker) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := tracker.base.RoundTrip(request)
	if err != nil {
		return response, err
	}
	if request.Method != http.MethodPost || !strings.Contains(request.URL.Path, "/ingest") {
		return response, nil
	}

	responseBody, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	response.Body = io.NopCloser(bytes.NewReader(responseBody))

	if response.StatusCode == http.StatusOK {
		tracker.requestCount.Add(1)
		tracker.ingestedEvents.Add(parseAxiomIngestedCount(responseBody))
	}
	return response, nil
}

func (tracker *axiomIngestTracker) stats() (ingestedEvents int, ingestRequests int) {
	return int(tracker.ingestedEvents.Load()), int(tracker.requestCount.Load())
}

func parseAxiomIngestedCount(body []byte) int32 {
	var ingestResponse struct {
		Ingested int32 `json:"ingested"`
	}
	if err := json.Unmarshal(body, &ingestResponse); err != nil {
		return 0
	}
	return ingestResponse.Ingested
}

func queryAxiomSpans(token, dataset, traceID string) ([]integrationSpanRecord, error) {
	startTime := time.Now().UTC().Add(-5 * time.Minute)
	endTime := time.Now().UTC().Add(time.Minute)

	query := fmt.Sprintf(
		"['%s'] | where trace_id == %q | project _time, name, trace_id, span_id, parent_span_id, kind, duration",
		dataset,
		traceID,
	)

	var lastErr error
	for attempt := 0; attempt < axiomPollAttempts; attempt++ {
		spans, err := runAxiomQuery(token, query, startTime, endTime)
		if err != nil {
			lastErr = err
		} else if len(spans) >= integrationSpanCount {
			return spans, nil
		} else {
			lastErr = fmt.Errorf("found %d spans, waiting for %d", len(spans), integrationSpanCount)
		}
		time.Sleep(axiomPollInterval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no spans found for trace_id %q", traceID)
	}
	return nil, lastErr
}

func runAxiomQuery(token, query string, startTime, endTime time.Time) ([]integrationSpanRecord, error) {
	requestBody, err := json.Marshal(map[string]string{
		"apl":       query,
		"startTime": startTime.Format(time.RFC3339Nano),
		"endTime":   endTime.Format(time.RFC3339Nano),
	})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, axiomQueryEndpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("status %d: %s (token may lack query permission)", response.StatusCode, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	return parseAxiomLegacyQuery(body)
}

func parseAxiomLegacyQuery(body []byte) ([]integrationSpanRecord, error) {
	var queryResponse struct {
		Matches []struct {
			Data map[string]json.RawMessage `json:"data"`
		} `json:"matches"`
	}
	if err := json.Unmarshal(body, &queryResponse); err != nil {
		return nil, fmt.Errorf("decode query response: %w", err)
	}

	spans := make([]integrationSpanRecord, 0, len(queryResponse.Matches))
	for _, match := range queryResponse.Matches {
		startTime, err := time.Parse(time.RFC3339Nano, decodeAxiomString(match.Data["_time"]))
		if err != nil {
			startTime, _ = time.Parse(time.RFC3339, decodeAxiomString(match.Data["_time"]))
		}

		span := integrationSpanRecord{
			Name:         decodeAxiomString(match.Data["name"]),
			TraceID:      decodeAxiomString(match.Data["trace_id"]),
			SpanID:       decodeAxiomString(match.Data["span_id"]),
			ParentSpanID: decodeAxiomString(match.Data["parent_span_id"]),
			StartTime:    startTime,
			Duration:     parseAxiomDurationRaw(match.Data["duration"]),
		}
		if span.Name == "" && span.TraceID == "" {
			continue
		}
		spans = append(spans, span)
	}
	return spans, nil
}

func decodeAxiomString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return strings.Trim(string(raw), `"`)
}

func parseAxiomDurationRaw(raw json.RawMessage) time.Duration {
	if len(raw) == 0 {
		return 0
	}
	var nanoseconds int64
	if err := json.Unmarshal(raw, &nanoseconds); err == nil {
		return time.Duration(nanoseconds)
	}
	var nanosecondsFloat float64
	if err := json.Unmarshal(raw, &nanosecondsFloat); err == nil {
		return time.Duration(nanosecondsFloat)
	}
	return parseAxiomDuration(decodeAxiomString(raw))
}

func parseAxiomDuration(value string) time.Duration {
	if value == "" {
		return 0
	}
	parsed, err := time.ParseDuration(value)
	if err == nil {
		return parsed
	}
	return 0
}
