package otlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

const (
	jsonContentType = "application/json"
	maxResponseBody = 64 * 1024
	errorBodyPrefix = 512
)

// HTTP posts OTLP JSON batches to an export endpoint.
type HTTP struct {
	endpoint       string
	headers        map[string]string
	httpClient     *http.Client
	requestContext context.Context
	cancel         context.CancelFunc
}

// NewHTTP creates an OTLP HTTP writer.
// endpoint is the full traces URL. headers are copied and may carry authorization.
func NewHTTP(endpoint string, headers map[string]string) *HTTP {
	requestContext, cancel := context.WithCancel(context.Background())
	return &HTTP{
		endpoint:       strings.TrimSpace(endpoint),
		headers:        cloneHeaders(headers),
		httpClient:     &http.Client{Timeout: ss.DefaultHTTPTimeout},
		requestContext: requestContext,
		cancel:         cancel,
	}
}

// WritePayload merges a batch of span documents and POSTs one JSON request.
func (writer *HTTP) WritePayload(payload []byte) error {
	merged, err := mergeTraceRequests(payload)
	if err != nil {
		return err
	}
	if len(merged) == 0 {
		return nil
	}

	request, err := http.NewRequestWithContext(writer.requestContext, http.MethodPost, writer.endpoint, bytes.NewReader(merged))
	if err != nil {
		return fmt.Errorf("create otlp request: %w", err)
	}
	for key, value := range writer.headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		request.Header.Set(key, value)
	}
	request.Header.Set("Content-Type", jsonContentType)

	response, err := writer.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send otlp request: %w", err)
	}
	defer response.Body.Close()

	body, readErr := readResponseBody(response.Body)
	if readErr != nil {
		return fmt.Errorf("read otlp response: %w", readErr)
	}
	if !ss.IsSuccessStatus(response.StatusCode) {
		return fmt.Errorf("otlp unexpected status %d: %s", response.StatusCode, bodyPrefix(body))
	}
	if rejected := rejectedSpanCount(body); rejected > 0 {
		return fmt.Errorf("otlp partial success: rejectedSpans=%d: %s", rejected, bodyPrefix(body))
	}
	return nil
}

// Close cancels in-flight requests and releases idle HTTP connections.
func (writer *HTTP) Close() error {
	if writer == nil {
		return nil
	}
	if writer.cancel != nil {
		writer.cancel()
	}
	if writer.httpClient != nil {
		writer.httpClient.CloseIdleConnections()
	}
	return nil
}

type otlpRequest struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

type otlpResourceSpans struct {
	Resource   json.RawMessage  `json:"resource,omitempty"`
	ScopeSpans []otlpScopeSpans `json:"scopeSpans"`
}

type otlpScopeSpans struct {
	Scope json.RawMessage   `json:"scope,omitempty"`
	Spans []json.RawMessage `json:"spans"`
}

// mergeTraceRequests combines newline-separated export documents into one request.
// The sender stores one encoded span per line. OTLP wants those spans in a single JSON body.
func mergeTraceRequests(payload []byte) ([]byte, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return nil, nil
	}

	var resource json.RawMessage
	var scope json.RawMessage
	var spans []json.RawMessage
	for _, line := range bytes.Split(payload, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var document otlpRequest
		if err := json.Unmarshal(line, &document); err != nil {
			return nil, fmt.Errorf("decode otlp batch: %w", err)
		}
		for _, resourceSpan := range document.ResourceSpans {
			if len(resource) == 0 && len(bytes.TrimSpace(resourceSpan.Resource)) > 0 {
				resource = append(json.RawMessage(nil), resourceSpan.Resource...)
			}
			for _, scopeSpan := range resourceSpan.ScopeSpans {
				if len(scope) == 0 && len(bytes.TrimSpace(scopeSpan.Scope)) > 0 {
					scope = append(json.RawMessage(nil), scopeSpan.Scope...)
				}
				spans = append(spans, scopeSpan.Spans...)
			}
		}
	}
	if len(spans) == 0 {
		return nil, nil
	}
	if len(resource) == 0 {
		resource = json.RawMessage("{}")
	}
	if len(scope) == 0 {
		scope = json.RawMessage(`{"name":"sseq"}`)
	}

	merged, err := marshalJSON(otlpRequest{
		ResourceSpans: []otlpResourceSpans{{
			Resource: resource,
			ScopeSpans: []otlpScopeSpans{{
				Scope: scope,
				Spans: spans,
			}},
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal otlp batch: %w", err)
	}
	return merged, nil
}

// readResponseBody returns the start of the body and discards the rest.
func readResponseBody(body io.Reader) ([]byte, error) {
	limited, err := io.ReadAll(io.LimitReader(body, maxResponseBody))
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(io.Discard, body)
	return limited, nil
}

// bodyPrefix returns the trimmed start of a response for an export error.
func bodyPrefix(body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > errorBodyPrefix {
		trimmed = trimmed[:errorBodyPrefix]
	}
	return string(trimmed)
}

// rejectedSpanCount reads partialSuccess.rejectedSpans, accepting a number or a decimal string.
// Jaeger 2.22 returns {"partialSuccess":{}} for a fully accepted export, which counts as zero.
func rejectedSpanCount(body []byte) int64 {
	if len(bytes.TrimSpace(body)) == 0 {
		return 0
	}
	var decoded struct {
		PartialSuccess *struct {
			RejectedSpans json.RawMessage `json:"rejectedSpans"`
		} `json:"partialSuccess"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil || decoded.PartialSuccess == nil {
		return 0
	}
	return parseRejectedSpans(decoded.PartialSuccess.RejectedSpans)
}

// parseRejectedSpans converts the proto JSON form of rejectedSpans to an integer.
func parseRejectedSpans(raw json.RawMessage) int64 {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return 0
	}
	literal := string(trimmed)
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return 0
		}
		literal = text
	}
	if parsed, err := strconv.ParseInt(literal, 10, 64); err == nil {
		return parsed
	}
	parsed, err := strconv.ParseFloat(literal, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed != math.Trunc(parsed) {
		return 0
	}
	return int64(parsed)
}

// cloneHeaders returns a copy of headers so the caller can reuse its map.
func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}
