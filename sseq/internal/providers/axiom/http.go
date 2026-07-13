package axiom

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
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
func (writer *HTTP) WritePayload(payload []byte) {
	request, err := http.NewRequest(http.MethodPost, writer.endpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: create axiom request: %v\n", err)
		return
	}
	request.Header.Set("Content-Type", ingestContentType)
	request.Header.Set("Authorization", "Bearer "+writer.token)

	response, err := writer.httpClient.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: send axiom request: %v\n", err)
		return
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "sseq: axiom unexpected status %d\n", response.StatusCode)
	}
}

// Close releases HTTP resources.
func (writer *HTTP) Close() error {
	return nil
}
