package axiom

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

const (
	defaultDomain     = "api.axiom.co"
	ingestContentType = "application/x-ndjson"
)

// HTTP posts Axiom NDJSON batches to the ingest API.
type HTTP struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

// NewHTTP creates an Axiom HTTP writer.
// domain may be empty (defaults to api.axiom.co). endpoint may override the ingest URL.
func NewHTTP(token, dataset, domain, endpoint string) (*HTTP, error) {
	if token == "" {
		return nil, fmt.Errorf("axiom requires token")
	}
	if dataset == "" {
		return nil, fmt.Errorf("axiom requires dataset")
	}

	resolvedDomain := strings.TrimSpace(domain)
	if resolvedDomain == "" {
		resolvedDomain = defaultDomain
	}
	resolvedDomain = strings.TrimPrefix(resolvedDomain, "https://")
	resolvedDomain = strings.TrimPrefix(resolvedDomain, "http://")
	resolvedDomain = strings.TrimSuffix(resolvedDomain, "/")

	resolvedEndpoint := strings.TrimSpace(endpoint)
	if resolvedEndpoint == "" {
		resolvedEndpoint = fmt.Sprintf("https://%s/v1/datasets/%s/ingest", resolvedDomain, dataset)
	}

	return &HTTP{
		endpoint:   resolvedEndpoint,
		token:      token,
		httpClient: &http.Client{Timeout: ss.DefaultHTTPTimeout},
	}, nil
}

// WritePayload delivers an NDJSON batch to Axiom.
func (writer *HTTP) WritePayload(payload []byte) error {
	request, err := http.NewRequest(http.MethodPost, writer.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create axiom request: %w", err)
	}
	request.Header.Set("Content-Type", ingestContentType)
	request.Header.Set("Authorization", "Bearer "+writer.token)

	response, err := writer.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send axiom request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if !ss.IsSuccessStatus(response.StatusCode) {
		return fmt.Errorf("axiom unexpected status %d", response.StatusCode)
	}
	return nil
}

// Close releases idle HTTP connections.
func (writer *HTTP) Close() error {
	if writer != nil && writer.httpClient != nil {
		writer.httpClient.CloseIdleConnections()
	}
	return nil
}
