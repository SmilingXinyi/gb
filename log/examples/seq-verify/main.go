package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const (
	seqUIEndpoint     = "http://localhost:5341"
	seqIngestEndpoint = "http://localhost:5342/ingest/clef"
	seqUsername       = "admin"
	seqPassword       = "Admin123456!"
	testApplication   = "gb-log-verify"
)

// clefEvent represents a Compact Log Event Format payload for Seq ingestion.
type clefEvent map[string]any

// spanSpec describes a span to emit for trace tree verification.
type spanSpec struct {
	spanID     string
	parentID   string
	message    string
	start      time.Time
	end        time.Time
	level      string
	properties map[string]any
}

func main() {
	traceID := mustTraceID()
	rootStart := time.Now().UTC().Add(-120 * time.Millisecond)

	spans := []spanSpec{
		{
			spanID:   "0123456789abc000",
			message:  "HTTP GET /api/users",
			start:    rootStart,
			end:      rootStart.Add(120 * time.Millisecond),
			level:    "Information",
			properties: map[string]any{
				"Application": testApplication,
				"@sk":         "Server",
				"http.method": "GET",
				"http.path":   "/api/users",
			},
		},
		{
			spanID:   "0123456789abc001",
			parentID: "0123456789abc000",
			message:  "Authenticate user",
			start:    rootStart.Add(10 * time.Millisecond),
			end:      rootStart.Add(40 * time.Millisecond),
			level:    "Debug",
			properties: map[string]any{
				"Application": testApplication,
				"@sk":         "Internal",
				"module":      "auth",
			},
		},
		{
			spanID:   "0123456789abc002",
			parentID: "0123456789abc000",
			message:  "Query users table",
			start:    rootStart.Add(50 * time.Millisecond),
			end:      rootStart.Add(100 * time.Millisecond),
			level:    "Debug",
			properties: map[string]any{
				"Application": testApplication,
				"@sk":         "Internal",
				"module":      "database",
				"sql":         "SELECT * FROM users LIMIT 10",
			},
		},
	}

	var payload bytes.Buffer
	for _, span := range spans {
		payload.Write(mustJSON(buildSpanEvent(traceID, span)))
		payload.WriteByte('\n')
	}

	// Correlated plain logs (no @st/@sp span semantics, only trace correlation).
	logEvents := []clefEvent{
		{
			"@t":            rootStart.Add(15 * time.Millisecond).Format(time.RFC3339Nano),
			"@l":            "Information",
			"@mt":           "Token validated for {User}",
			"User":          "alice",
			"Application":   testApplication,
			"module":        "auth",
			"@tr":           traceID,
			"@sp":           "0123456789abc001",
			"verify_marker":      "gb-log-seq-verify",
			"verify_log_marker":  "gb-log-seq-log",
		},
		{
			"@t":            rootStart.Add(60 * time.Millisecond).Format(time.RFC3339Nano),
			"@l":            "Information",
			"@mt":           "Fetched {Count} users from database",
			"Count":         10,
			"Application":   testApplication,
			"module":        "database",
			"@tr":           traceID,
			"@sp":           "0123456789abc002",
			"verify_marker":      "gb-log-seq-verify",
			"verify_log_marker":  "gb-log-seq-log",
		},
	}
	for _, event := range logEvents {
		payload.Write(mustJSON(event))
		payload.WriteByte('\n')
	}

	if err := ingestCLEF(payload.Bytes()); err != nil {
		panic(fmt.Sprintf("ingest failed: %v", err))
	}
	fmt.Println("✓ Ingested span tree and correlated logs")

	client, csrfToken, err := loginSeqClient()
	if err != nil {
		panic(fmt.Sprintf("login failed: %v", err))
	}

	spanEvents, err := queryEvents(client, csrfToken,
		fmt.Sprintf("verify_marker = 'gb-log-seq-verify' and Has(@Start) and Has(@SpanId) and @TraceId = '%s'", traceID))
	if err != nil {
		panic(fmt.Sprintf("query spans failed: %v", err))
	}

	logEventsResult, err := queryEvents(client, csrfToken,
		fmt.Sprintf("verify_log_marker = 'gb-log-seq-log' and @TraceId = '%s'", traceID))
	if err != nil {
		panic(fmt.Sprintf("query logs failed: %v", err))
	}

	fmt.Printf("✓ Seq indexed %d span event(s) and %d correlated log event(s)\n", len(spanEvents), len(logEventsResult))

	if len(spanEvents) != 3 {
		panic(fmt.Sprintf("expected 3 spans, got %d", len(spanEvents)))
	}

	printSpanTree(spanEvents)

	if err := verifySpanHierarchy(spanEvents); err != nil {
		panic(err)
	}
	fmt.Println("✓ Span hierarchy matches expected parent-child relationships")

	fmt.Println()
	fmt.Println("Web UI verification:")
	fmt.Printf("  1. Open %s\n", seqUIEndpoint)
	fmt.Printf("  2. Login: %s / %s\n", seqUsername, seqPassword)
	fmt.Printf("  3. Search: @TraceId = '%s'\n", traceID)
	fmt.Println("  4. Click any span duration bar or use Trace → Show to open the trace timeline")
	fmt.Println("  5. Expected tree:")
	fmt.Println("     HTTP GET /api/users  (root, ~120ms)")
	fmt.Println("     ├── Authenticate user (~30ms)")
	fmt.Println("     └── Query users table (~50ms)")
}

// buildSpanEvent converts a spanSpec into a CLEF span event for Seq.
func buildSpanEvent(traceID string, span spanSpec) clefEvent {
	event := clefEvent{
		"@t":            span.end.Format(time.RFC3339Nano),
		"@st":           span.start.Format(time.RFC3339Nano),
		"@tr":           traceID,
		"@sp":           span.spanID,
		"@mt":           span.message,
		"@l":            span.level,
		"verify_marker": "gb-log-seq-verify",
	}
	if span.parentID != "" {
		event["@ps"] = span.parentID
	}
	for key, value := range span.properties {
		event[key] = value
	}
	return event
}

// ingestCLEF posts newline-delimited CLEF events to Seq.
func ingestCLEF(body []byte) error {
	request, err := http.NewRequest(http.MethodPost, seqIngestEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/vnd.serilog.clef")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	responseBody, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return nil
}

// loginSeqClient authenticates against Seq and returns an HTTP client with session cookies.
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

// seqEvent mirrors the subset of Seq event fields needed for verification.
type seqEvent struct {
	RenderedMessage string `json:"RenderedMessage"`
	TraceId         string `json:"TraceId"`
	SpanId          string `json:"SpanId"`
	ParentId        string `json:"ParentId"`
	Start           string `json:"Start"`
	Elapsed         string `json:"Elapsed"`
}

// queryEvents retrieves rendered events from Seq using a filter expression.
func queryEvents(client *http.Client, csrfToken, filter string) ([]seqEvent, error) {
	query := url.Values{}
	query.Set("count", "100")
	query.Set("render", "true")
	query.Set("filter", filter)
	requestURL := seqUIEndpoint + "/api/events?" + query.Encode()
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
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

	var events []seqEvent
	if err := json.NewDecoder(response.Body).Decode(&events); err != nil {
		return nil, err
	}
	return events, nil
}

// verifySpanHierarchy checks that parent-child span relationships are preserved by Seq.
func verifySpanHierarchy(events []seqEvent) error {
	spanByID := make(map[string]seqEvent, len(events))
	for _, event := range events {
		if event.SpanId == "" || event.Start == "" {
			return fmt.Errorf("event %q is missing @SpanId or @Start", event.RenderedMessage)
		}
		spanByID[event.SpanId] = event
	}

	expectedParents := map[string]string{
		"0123456789abc001": "0123456789abc000",
		"0123456789abc002": "0123456789abc000",
	}
	for spanID, expectedParent := range expectedParents {
		event, ok := spanByID[spanID]
		if !ok {
			return fmt.Errorf("missing span %s", spanID)
		}
		if event.ParentId != expectedParent {
			return fmt.Errorf("span %s parent = %q, want %q", spanID, event.ParentId, expectedParent)
		}
	}

	root, ok := spanByID["0123456789abc000"]
	if !ok {
		return fmt.Errorf("missing root span")
	}
	if root.ParentId != "" {
		return fmt.Errorf("root span should not have a parent, got %q", root.ParentId)
	}
	return nil
}

// printSpanTree prints span events in a human-readable tree layout.
func printSpanTree(events []seqEvent) {
	fmt.Println()
	fmt.Println("Indexed spans:")
	for _, event := range events {
		parent := event.ParentId
		if parent == "" {
			parent = "(root)"
		}
		fmt.Printf("  - %s | span=%s parent=%s\n", event.RenderedMessage, event.SpanId, parent)
	}
}

func mustTraceID() string {
	var randomBytes [16]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(randomBytes[:])
}

func mustJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
