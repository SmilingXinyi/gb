package sender

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

const seqCLEFContentType = "application/vnd.serilog.clef"

// HTTPConfig defines Seq HTTP ingestion settings.
type HTTPConfig struct {
	Endpoint   string
	APIKey     string
	HTTPClient *http.Client
}

// HTTPProvider posts CLEF batches to a Seq ingestion endpoint.
type HTTPProvider struct {
	config     HTTPConfig
	httpClient *http.Client
}

// NewHTTPProvider creates a Seq HTTP payload writer.
func NewHTTPProvider(config HTTPConfig) *HTTPProvider {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &HTTPProvider{
		config:     config,
		httpClient: httpClient,
	}
}

// WritePayload delivers a CLEF batch to Seq and logs transport failures to stderr.
func (provider *HTTPProvider) WritePayload(payload []byte) {
	request, err := http.NewRequest(http.MethodPost, provider.config.Endpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: create request: %v\n", err)
		return
	}

	request.Header.Set("Content-Type", seqCLEFContentType)
	if provider.config.APIKey != "" {
		request.Header.Set("X-Seq-ApiKey", provider.config.APIKey)
	}

	response, err := provider.httpClient.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: send request: %v\n", err)
		return
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	if response.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "sseq: unexpected status %d\n", response.StatusCode)
	}
}

// Close releases HTTP provider resources.
func (provider *HTTPProvider) Close() error {
	return nil
}
