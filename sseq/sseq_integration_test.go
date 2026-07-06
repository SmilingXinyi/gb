package sseq_test

import (
	"bytes"
	"context"
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
		Application:   "sseq-integration",
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
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

	spans, err := querySeqSpans(traceID)
	if err != nil {
		t.Fatalf("query seq spans: %v", err)
	}

	spanByMessage := make(map[string]seqSpan)
	for _, span := range spans {
		spanByMessage[span.Message] = span
	}

	if len(spanByMessage) < 3 {
		t.Fatalf("expected 3 spans, got %d: %+v", len(spanByMessage), spans)
	}

	root, ok := spanByMessage["HTTP GET /api/users"]
	if !ok {
		t.Fatalf("missing root span in %+v", spanByMessage)
	}
	if root.ParentID != "" {
		t.Fatalf("root span should not have parent, got %q", root.ParentID)
	}
	if root.Start == "" || root.Elapsed == "" {
		t.Fatal("root span missing timeline fields")
	}

	for _, name := range []string{"Authenticate user", "Query users table"} {
		child, found := spanByMessage[name]
		if !found {
			t.Fatalf("missing child span %q", name)
		}
		if child.ParentID != root.SpanID {
			t.Fatalf("span %q parent = %q, want %q", name, child.ParentID, root.SpanID)
		}
	}
}

type seqSpan struct {
	Message     string `json:"RenderedMessage"`
	TraceID     string `json:"TraceId"`
	SpanID      string `json:"SpanId"`
	ParentID    string `json:"ParentId"`
	Start       string `json:"Start"`
	Elapsed     string `json:"Elapsed"`
	Application string `json:"Application"`
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

func querySeqSpans(traceID string) ([]seqSpan, error) {
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

	var spans []seqSpan
	if err := json.NewDecoder(response.Body).Decode(&spans); err != nil {
		return nil, err
	}
	return spans, nil
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
