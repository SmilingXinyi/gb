package sseq_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

	sseq.Setup(sseq.Config{
		Provider:      sseq.ProviderAxiom,
		Application:   "sseq-integration",
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		Axiom: sseq.AxiomConfig{
			Token:   token,
			Dataset: dataset,
			Domain:  "api.axiom.co",
		},
	})

	requestContext, rootSpan := sseq.Start(context.Background(), "HTTP GET /api/users")
	traceID := rootSpan.TraceID()

	err := sseq.Do(requestContext, "Authenticate user", func(ctx context.Context) error {
		time.Sleep(15 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("auth span error = %v", err)
	}

	err = sseq.Do(requestContext, "Query users table", func(ctx context.Context) error {
		time.Sleep(25 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("db span error = %v", err)
	}

	rootSpan.End()
	sseq.Shutdown()

	if traceID == "" {
		t.Fatal("expected trace id from root span")
	}

	spans, err := queryAxiomSpans(token, dataset, traceID)
	if err != nil {
		if strings.Contains(err.Error(), "token may lack query permission") {
			t.Skipf("ingest completed but query verification skipped: %v", err)
		}
		t.Fatalf("query axiom spans: %v", err)
	}

	spanByName := make(map[string]axiomSpan, len(spans))
	for _, span := range spans {
		spanByName[span.Name] = span
	}

	if len(spanByName) < 3 {
		t.Fatalf("expected 3 spans, got %d: %+v", len(spanByName), spans)
	}

	root, ok := spanByName["HTTP GET /api/users"]
	if !ok {
		t.Fatalf("missing root span in %+v", spanByName)
	}
	if root.ParentSpanID != "" {
		t.Fatalf("root span should not have parent_span_id, got %q", root.ParentSpanID)
	}
	if root.ServiceName != "sseq-integration" {
		t.Fatalf("service.name = %q, want sseq-integration", root.ServiceName)
	}

	for _, name := range []string{"Authenticate user", "Query users table"} {
		child, found := spanByName[name]
		if !found {
			t.Fatalf("missing child span %q", name)
		}
		if child.ParentSpanID != root.SpanID {
			t.Fatalf("span %q parent_span_id = %q, want %q", name, child.ParentSpanID, root.SpanID)
		}
		if child.TraceID != traceID {
			t.Fatalf("span %q trace_id = %q, want %q", name, child.TraceID, traceID)
		}
	}
}

type axiomSpan struct {
	Name         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	ServiceName  string
	Kind         string
}

func queryAxiomSpans(token, dataset, traceID string) ([]axiomSpan, error) {
	startTime := time.Now().UTC().Add(-5 * time.Minute)
	endTime := time.Now().UTC().Add(time.Minute)

	query := fmt.Sprintf(
		"['%s'] | where trace_id == %q | project name, trace_id, span_id, parent_span_id, kind, ['service.name']",
		dataset,
		traceID,
	)

	var lastErr error
	for attempt := 0; attempt < axiomPollAttempts; attempt++ {
		spans, err := runAxiomQuery(token, query, startTime, endTime)
		if err != nil {
			lastErr = err
		} else if len(spans) >= 3 {
			return spans, nil
		} else {
			lastErr = fmt.Errorf("found %d spans, waiting for 3", len(spans))
		}
		time.Sleep(axiomPollInterval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no spans found for trace_id %q", traceID)
	}
	return nil, lastErr
}

func runAxiomQuery(token, query string, startTime, endTime time.Time) ([]axiomSpan, error) {
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

func parseAxiomLegacyQuery(body []byte) ([]axiomSpan, error) {
	var queryResponse struct {
		Matches []struct {
			Data map[string]json.RawMessage `json:"data"`
		} `json:"matches"`
		Status struct {
			RowsMatched int64  `json:"rowsMatched"`
			ElapsedTime string `json:"elapsedTime"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &queryResponse); err != nil {
		return nil, fmt.Errorf("decode query response: %w", err)
	}

	spans := make([]axiomSpan, 0, len(queryResponse.Matches))
	for _, match := range queryResponse.Matches {
		span := axiomSpan{
			Name:         decodeAxiomString(match.Data["name"]),
			TraceID:      decodeAxiomString(match.Data["trace_id"]),
			SpanID:       decodeAxiomString(match.Data["span_id"]),
			ParentSpanID: decodeAxiomString(match.Data["parent_span_id"]),
			ServiceName:  decodeAxiomString(match.Data["service.name"]),
			Kind:         decodeAxiomString(match.Data["kind"]),
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
