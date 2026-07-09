package middleware

import (
	"fmt"
	"net/http"

	"github.com/SmilingXinyi/gb/sseq"
)

// HTTP wraps an http.Handler and records each request as a root span.
func HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		spanName := fmt.Sprintf("%s %s", request.Method, request.URL.Path)
		requestContext, span := sseq.Start(request.Context(), spanName)
		defer span.End()

		recorder := &statusRecorder{
			responseWriter: response,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(recorder, request.WithContext(requestContext))
		span.SetHTTPStatusCode(recorder.statusCode)
	})
}

type statusRecorder struct {
	responseWriter http.ResponseWriter
	statusCode     int
}

func (recorder *statusRecorder) Header() http.Header {
	return recorder.responseWriter.Header()
}

func (recorder *statusRecorder) Write(body []byte) (int, error) {
	return recorder.responseWriter.Write(body)
}

func (recorder *statusRecorder) WriteHeader(statusCode int) {
	recorder.statusCode = statusCode
	recorder.responseWriter.WriteHeader(statusCode)
}
