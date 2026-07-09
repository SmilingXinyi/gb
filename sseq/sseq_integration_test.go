package sseq_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
)

const (
	seqUIEndpoint     = "http://localhost:5341"
	seqIngestEndpoint = "http://localhost:5342/ingest/clef"
	seqUsername       = "admin"
	seqPassword       = "Admin123456!"
)

func TestIntegrationSpanTreeWithSeqDocker(t *testing.T) {
	if os.Getenv("SSEQ_SKIP_INTEGRATION") == "1" {
		t.Skip("integration disabled by SSEQ_SKIP_INTEGRATION=1")
	}

	if !seqAvailable(t) {
		t.Skip("Seq docker service is not available")
	}

	sseq.Setup(sseq.Config{
		Endpoint:      seqIngestEndpoint,
		Application:   integrationApplication,
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
	})
	t.Cleanup(sseq.Shutdown)

	traceID, err := runIntegrationSpanScenario()
	if err != nil {
		t.Fatalf("runIntegrationSpanScenario() error = %v", err)
	}
	if traceID == "" {
		t.Fatal("expected trace id from root span")
	}

	sseq.Shutdown()

	spans, err := querySeqSpans(traceID)
	if err != nil {
		t.Fatalf("query seq spans: %v", err)
	}

	verifyIntegrationSpanTree(t, traceID, spans)
}

func seqAvailable(t *testing.T) bool {
	t.Helper()

	response, err := http.Get(seqUIEndpoint)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}

func querySeqSpans(traceID string) ([]integrationSpanRecord, error) {
	client, csrfToken, err := loginSeqClient()
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("count", "100")
	query.Set("render", "true")
	query.Set("filter", "Has(@Start) and Has(@SpanId) and @TraceId = '"+traceID+"'")

	request, err := http.NewRequest(http.MethodGet, seqUIEndpoint+"/api/events?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Seq-CsrfToken", csrfToken)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var seqSpans []seqSpan
	if err := json.NewDecoder(response.Body).Decode(&seqSpans); err != nil {
		return nil, err
	}

	spans := make([]integrationSpanRecord, 0, len(seqSpans))
	for _, seqSpan := range seqSpans {
		startTime, err := time.Parse(time.RFC3339Nano, seqSpan.Start)
		if err != nil {
			startTime, _ = time.Parse(time.RFC3339, seqSpan.Start)
		}
		spans = append(spans, integrationSpanRecord{
			Name:         seqSpan.Message,
			TraceID:      seqSpan.TraceID,
			SpanID:       seqSpan.SpanID,
			ParentSpanID: seqSpan.ParentID,
			StartTime:    startTime,
			Duration:     parseSeqElapsed(seqSpan.Elapsed),
		})
	}
	return spans, nil
}

type seqSpan struct {
	Message  string `json:"RenderedMessage"`
	TraceID  string `json:"TraceId"`
	SpanID   string `json:"SpanId"`
	ParentID string `json:"ParentId"`
	Start    string `json:"Start"`
	Elapsed  string `json:"Elapsed"`
}

func parseSeqElapsed(value string) time.Duration {
	if value == "" {
		return 0
	}
	parsed, err := time.ParseDuration(value)
	if err == nil {
		return parsed
	}
	return 0
}

func loginSeqClient() (*http.Client, string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, "", err
	}
	client := &http.Client{Jar: jar}

	loginBody, err := json.Marshal(map[string]string{
		"Username": seqUsername,
		"Password": seqPassword,
	})
	if err != nil {
		return nil, "", err
	}

	request, err := http.NewRequest(http.MethodPost, seqUIEndpoint+"/api/users/login", bytes.NewReader(loginBody))
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	var loginResponse struct {
		CsrfToken string `json:"CsrfToken"`
		Error     string `json:"Error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&loginResponse); err != nil {
		return nil, "", err
	}
	if loginResponse.Error != "" {
		return nil, "", fmt.Errorf("%s", loginResponse.Error)
	}
	return client, loginResponse.CsrfToken, nil
}
