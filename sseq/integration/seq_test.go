package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
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
	// Seq 2025 requires a first-login password change; use a distinct value.
	seqPasswordAfterChange = "Admin123456!Changed"
)

func TestIntegrationSpanTreeWithSeqDocker(t *testing.T) {
	if os.Getenv("SSEQ_SKIP_INTEGRATION") == "1" {
		t.Skip("integration disabled by SSEQ_SKIP_INTEGRATION=1")
	}

	if !seqAvailable(t) {
		t.Skip("Seq docker service is not available")
	}

	if err := sseq.SetupSeq(seqIngestEndpoint, "", integrationApplication); err != nil {
		t.Fatalf("SetupSeq() error = %v", err)
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
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}

	// Seq renders Elapsed as a .NET TimeSpan, e.g. "00:00:00.0908570".
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0
	}
	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds*float64(time.Second))
}

func loginSeqClient() (*http.Client, string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, "", err
	}
	client := &http.Client{Jar: jar}

	csrfToken, mustChangePassword, err := postSeqLogin(client, seqPassword, "")
	if err == nil {
		return client, csrfToken, nil
	}
	if mustChangePassword {
		csrfToken, _, err = postSeqLogin(client, seqPassword, seqPasswordAfterChange)
		if err != nil {
			return nil, "", err
		}
		return client, csrfToken, nil
	}

	// Password may already have been rotated by a previous test run.
	csrfToken, _, err = postSeqLogin(client, seqPasswordAfterChange, "")
	if err != nil {
		return nil, "", err
	}
	return client, csrfToken, nil
}

// postSeqLogin authenticates against Seq. When newPassword is set, Seq performs
// the required first-login password change in the same request.
func postSeqLogin(client *http.Client, password, newPassword string) (csrfToken string, mustChangePassword bool, err error) {
	payload := map[string]string{
		"Username": seqUsername,
		"Password": password,
	}
	if newPassword != "" {
		payload["NewPassword"] = newPassword
	}

	loginBody, err := json.Marshal(payload)
	if err != nil {
		return "", false, err
	}

	request, err := http.NewRequest(http.MethodPost, seqUIEndpoint+"/api/users/login", bytes.NewReader(loginBody))
	if err != nil {
		return "", false, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return "", false, err
	}
	defer response.Body.Close()

	var loginResponse struct {
		CsrfToken         string `json:"CsrfToken"`
		Error              string `json:"Error"`
		MustChangePassword bool   `json:"MustChangePassword"`
	}
	if err := json.NewDecoder(response.Body).Decode(&loginResponse); err != nil {
		return "", false, err
	}
	if loginResponse.Error != "" {
		return "", loginResponse.MustChangePassword, fmt.Errorf("%s", loginResponse.Error)
	}
	return loginResponse.CsrfToken, false, nil
}
