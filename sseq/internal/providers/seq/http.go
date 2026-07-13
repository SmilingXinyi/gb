package seq

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

const clefContentType = "application/vnd.serilog.clef"

// HTTP posts CLEF batches to a Seq ingestion endpoint.
type HTTP struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

// NewHTTP creates a Seq HTTP writer.
func NewHTTP(endpoint, apiKey string) *HTTP {
	return &HTTP{
		endpoint:   endpoint,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: ss.DefaultHTTPTimeout},
	}
}

// WritePayload delivers a CLEF batch to Seq.
func (writer *HTTP) WritePayload(payload []byte) {
	request, err := http.NewRequest(http.MethodPost, writer.endpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: create seq request: %v\n", err)
		return
	}
	request.Header.Set("Content-Type", clefContentType)
	if writer.apiKey != "" {
		request.Header.Set("X-Seq-ApiKey", writer.apiKey)
	}

	response, err := writer.httpClient.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sseq: send seq request: %v\n", err)
		return
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "sseq: seq unexpected status %d\n", response.StatusCode)
	}
}

// Close releases HTTP resources.
func (writer *HTTP) Close() error {
	return nil
}
