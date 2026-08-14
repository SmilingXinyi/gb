package seq

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

const clefContentType = "application/vnd.serilog.clef"

// HTTP posts CLEF batches to a Seq ingestion endpoint.
type HTTP struct {
	endpoint       string
	apiKey         string
	httpClient     *http.Client
	requestContext context.Context
	cancel         context.CancelFunc
}

// NewHTTP creates a Seq HTTP writer.
func NewHTTP(endpoint, apiKey string) *HTTP {
	requestContext, cancel := context.WithCancel(context.Background())
	return &HTTP{
		endpoint:       endpoint,
		apiKey:         apiKey,
		httpClient:     &http.Client{Timeout: ss.DefaultHTTPTimeout},
		requestContext: requestContext,
		cancel:         cancel,
	}
}

// WritePayload delivers a CLEF batch to Seq.
func (writer *HTTP) WritePayload(payload []byte) error {
	request, err := http.NewRequestWithContext(writer.requestContext, http.MethodPost, writer.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create seq request: %w", err)
	}
	request.Header.Set("Content-Type", clefContentType)
	if writer.apiKey != "" {
		request.Header.Set("X-Seq-ApiKey", writer.apiKey)
	}

	response, err := writer.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send seq request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if !ss.IsSuccessStatus(response.StatusCode) {
		return fmt.Errorf("seq unexpected status %d", response.StatusCode)
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
