package sender

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultAxiomDomain       = "api.axiom.co"
	axiomIngestContentType   = "application/x-ndjson"
	axiomAuthorizationPrefix = "Bearer "
)

// AxiomConfig defines direct Axiom ingest settings.
type AxiomConfig struct {
	Token      string
	Dataset    string
	Domain     string
	Endpoint   string
	HTTPClient *http.Client
}

// AxiomBatchConfig defines Axiom provider settings with batching defaults.
type AxiomBatchConfig struct {
	Axiom         AxiomConfig
	BatchSize     int
	FlushInterval time.Duration
}

// AxiomProvider posts Axiom-compatible NDJSON batches to the ingest API.
type AxiomProvider struct {
	config     AxiomConfig
	endpoint   string
	httpClient *http.Client
}

// NewAxiomProvider creates an Axiom ingest payload writer.
func NewAxiomProvider(config AxiomConfig) (*AxiomProvider, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("axiom provider requires token")
	}
	if config.Dataset == "" {
		return nil, fmt.Errorf("axiom provider requires dataset")
	}

	domain := strings.TrimSpace(config.Domain)
	if domain == "" {
		domain = defaultAxiomDomain
	}
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	endpoint := strings.TrimSpace(config.Endpoint)
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s/v1/ingest/%s", domain, config.Dataset)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &AxiomProvider{
		config:     config,
		endpoint:   endpoint,
		httpClient: httpClient,
	}, nil
}

// WritePayload delivers an NDJSON batch to Axiom and logs transport failures to stderr.
func (provider *AxiomProvider) WritePayload(payload []byte) {
	request, err := http.NewRequest(http.MethodPost, provider.endpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: create axiom request: %v\n", err)
		return
	}

	request.Header.Set("Content-Type", axiomIngestContentType)
	request.Header.Set("Authorization", axiomAuthorizationPrefix+provider.config.Token)

	response, err := provider.httpClient.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: send axiom request: %v\n", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "sseq: axiom unexpected status %d\n", response.StatusCode)
	}
}

// Close releases Axiom provider resources.
func (provider *AxiomProvider) Close() error {
	return nil
}
