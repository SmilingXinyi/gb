package otlp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

func TestHTTPMergesBatchIntoOneJSONRequest(t *testing.T) {
	var bodies [][]byte
	var contentTypes []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		bodies = append(bodies, body)
		contentTypes = append(contentTypes, request.Header.Get("Content-Type"))
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte(`{"partialSuccess":{}}`))
	}))
	defer server.Close()

	encoder := NewEncoder("api", map[string]any{"deployment.environment": "test"})
	first, err := encoder.Encode(ss.SpanEvent{
		Name:      "checkout",
		TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:    "bbbbbbbbbbbbbbbb",
		StartTime: time.Unix(1, 0).UTC(),
		EndTime:   time.Unix(3, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	second, err := encoder.Encode(ss.SpanEvent{
		Name:      "charge",
		TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:    "cccccccccccccccc",
		ParentID:  "bbbbbbbbbbbbbbbb",
		StartTime: time.Unix(2, 0).UTC(),
		EndTime:   time.Unix(3, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	writer := NewHTTP(server.URL, map[string]string{
		"Authorization": "Bearer secret",
		"Content-Type":  "text/plain",
	})
	payload := append(append(append([]byte{}, first...), '\n'), second...)
	payload = append(payload, '\n')
	if err := writer.WritePayload(payload); err != nil {
		t.Fatalf("WritePayload() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if len(bodies) != 1 {
		t.Fatalf("requests = %d", len(bodies))
	}
	if contentTypes[0] != jsonContentType {
		t.Fatalf("Content-Type = %q", contentTypes[0])
	}

	decoder := json.NewDecoder(bytes.NewReader(bodies[0]))
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("Decode() error = %v body=%s", err, bodies[0])
	}
	if decoder.More() {
		t.Fatal("batch was posted as more than one JSON document")
	}
	_, _, spans := decodeDocument(t, bodies[0])
	if len(spans) != 2 || spans[0]["name"] != "checkout" || spans[1]["name"] != "charge" {
		t.Fatalf("spans = %#v", spans)
	}
	if spans[1]["parentSpanId"] != "bbbbbbbbbbbbbbbb" {
		t.Fatalf("parentSpanId = %v", spans[1]["parentSpanId"])
	}
}

func TestHTTPSkipsEmptyPayload(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		called = true
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer := NewHTTP(server.URL, nil)
	if err := writer.WritePayload(nil); err != nil {
		t.Fatalf("WritePayload(nil) error = %v", err)
	}
	if err := writer.WritePayload([]byte("\n\n")); err != nil {
		t.Fatalf("WritePayload(blank) error = %v", err)
	}
	if called {
		t.Fatal("empty payload was sent")
	}
}

func TestHTTPReportsStatusAndPartialSuccess(t *testing.T) {
	longBody := "HEAD-" + strings.Repeat("x", 700) + "TAIL"
	tests := []struct {
		name      string
		status    int
		body      string
		wantError string
		absent    string
	}{
		{name: "empty success", status: http.StatusOK, body: ""},
		{name: "created", status: http.StatusCreated, body: "{}"},
		{name: "jaeger empty partial success", status: http.StatusOK, body: `{"partialSuccess":{}}`},
		{name: "rejected zero", status: http.StatusOK, body: `{"partialSuccess":{"rejectedSpans":0}}`},
		{name: "rejected zero string", status: http.StatusOK, body: `{"partialSuccess":{"rejectedSpans":"0"}}`},
		{name: "not json success", status: http.StatusOK, body: "ok"},
		{
			name:      "rejected number",
			status:    http.StatusOK,
			body:      `{"partialSuccess":{"rejectedSpans":2,"errorMessage":"bad span"}}`,
			wantError: "rejectedSpans=2",
		},
		{
			name:      "rejected string",
			status:    http.StatusOK,
			body:      `{"partialSuccess":{"rejectedSpans":"3","errorMessage":"span rejected"}}`,
			wantError: "otlp partial success: rejectedSpans=3",
		},
		{
			name:      "status",
			status:    http.StatusUnsupportedMediaType,
			body:      "unsupported media type",
			wantError: "otlp unexpected status 415: unsupported media type",
		},
		{
			name:      "truncated body",
			status:    http.StatusInternalServerError,
			body:      longBody,
			wantError: "otlp unexpected status 500: HEAD-",
			absent:    "TAIL",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.WriteHeader(test.status)
				_, _ = response.Write([]byte(test.body))
			}))
			defer server.Close()

			writer := NewHTTP(server.URL, nil)
			err := writer.WritePayload(sampleBatch(t))
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("WritePayload() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("WritePayload() error = %v, want substring %q", err, test.wantError)
			}
			if test.absent != "" && strings.Contains(err.Error(), test.absent) {
				t.Fatalf("error includes body tail: %v", err)
			}
		})
	}
}

// sampleBatch is one encoded span, which is what the sender flushes as a line.
func sampleBatch(t *testing.T) []byte {
	t.Helper()
	payload, err := Encoder{}.Encode(ss.SpanEvent{
		Name:      "ping",
		TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:    "bbbbbbbbbbbbbbbb",
		StartTime: time.Unix(1, 0).UTC(),
		EndTime:   time.Unix(2, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	return append(payload, '\n')
}

func TestHTTPRejectsInvalidBatch(t *testing.T) {
	writer := NewHTTP("http://127.0.0.1", nil)
	err := writer.WritePayload([]byte("{not-json}\n"))
	if err == nil || !strings.Contains(err.Error(), "decode otlp batch") {
		t.Fatalf("WritePayload() error = %v", err)
	}
}
