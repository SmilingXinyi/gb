package middleware

import (
	"fmt"
	"net/http"

	"github.com/SmilingXinyi/gb/sseq"
)

// HTTP wraps an http.Handler and records each request as a root span in Seq.
func HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		spanName := fmt.Sprintf("%s %s", request.Method, request.URL.Path)
		requestContext, span := sseq.Start(request.Context(), spanName)
		defer span.End()
		next.ServeHTTP(response, request.WithContext(requestContext))
	})
}
