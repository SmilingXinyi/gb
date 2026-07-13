package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
	"github.com/joho/godotenv"
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

	// Load local credentials from sseq/.env when present (does not override existing env).
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	token := os.Getenv("AXIOM_TOKEN")
	dataset := os.Getenv("AXIOM_DATASET")
	if token == "" || dataset == "" {
		t.Skip("AXIOM_TOKEN and AXIOM_DATASET are required for Axiom integration test")
	}

	if err := sseq.SetupAxiom(token, dataset, integrationApplication); err != nil {
		t.Fatalf("SetupAxiom() error = %v", err)
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
	if queryErr != nil {
		t.Fatalf("query axiom spans: %v", queryErr)
	}
	verifyIntegrationSpanTree(t, traceID, spans)
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
