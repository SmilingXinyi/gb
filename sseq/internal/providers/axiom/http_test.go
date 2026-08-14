package axiom

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPRejectsMissingCredentials(t *testing.T) {
	if _, err := NewHTTP("", "dataset", "", ""); err == nil {
		t.Fatal("expected token error")
	}
	if _, err := NewHTTP("token", "", "", ""); err == nil {
		t.Fatal("expected dataset error")
	}
}

func TestHTTPWritePayloadSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer, err := NewHTTP("token", "dataset", "", server.URL)
	if err != nil {
		t.Fatalf("NewHTTP() error = %v", err)
	}
	if err := writer.WritePayload([]byte("{}\n")); err != nil {
		t.Fatalf("WritePayload() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
