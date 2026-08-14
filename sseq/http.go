package sseq

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const (
	traceparentHeader  = "traceparent"
	traceparentVersion = "00"
	traceparentFlags   = "01"
)

// HTTP wraps an http.Handler and records each request as a server span.
// Incoming W3C traceparent headers continue the remote trace.
func HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestContext := request.Context()
		if traceID, parentSpanID, ok := Extract(request.Header); ok {
			requestContext = Resume(requestContext, traceID, parentSpanID)
		}

		spanName := request.Method
		if spanName == "" {
			spanName = "HTTP"
		}
		requestContext, span := defaultTracer().Start(requestContext, spanName, "server")
		span.Set("http.method", request.Method)
		span.Set("http.target", request.URL.RequestURI())
		if request.Host != "" {
			span.Set("http.host", request.Host)
		}

		wrapped, recorder := wrapResponseWriter(response)
		defer func() {
			if recovered := recover(); recovered != nil {
				span.RecordError(fmt.Errorf("panic: %v", recovered))
				span.End()
				panic(recovered)
			}
			span.SetHTTPStatus(recorder.statusCode)
			span.End()
		}()

		next.ServeHTTP(wrapped, request.WithContext(requestContext))
	})
}

// Inject writes a W3C traceparent header from the active span in ctx.
func Inject(ctx context.Context, header http.Header) {
	if header == nil {
		return
	}
	traceID, spanID, ok := IDs(ctx)
	if !ok {
		return
	}
	traceID = strings.ToLower(traceID)
	spanID = strings.ToLower(spanID)
	if !isHex(traceID, 32) || !isHex(spanID, 16) || isAllZero(traceID) || isAllZero(spanID) {
		return
	}
	header.Set(traceparentHeader, traceparentVersion+"-"+traceID+"-"+spanID+"-"+traceparentFlags)
}

// Extract reads a W3C traceparent header.
func Extract(header http.Header) (traceID, spanID string, ok bool) {
	if header == nil {
		return "", "", false
	}
	value := strings.TrimSpace(header.Get(traceparentHeader))
	if value == "" {
		return "", "", false
	}
	parts := strings.Split(value, "-")
	if len(parts) != 4 {
		return "", "", false
	}
	version := strings.ToLower(parts[0])
	headerTraceID := strings.ToLower(parts[1])
	headerSpanID := strings.ToLower(parts[2])
	flags := strings.ToLower(parts[3])
	if !isHex(version, 2) || !isHex(headerTraceID, 32) || !isHex(headerSpanID, 16) || !isHex(flags, 2) {
		return "", "", false
	}
	if isAllZero(headerTraceID) || isAllZero(headerSpanID) {
		return "", "", false
	}
	return headerTraceID, headerSpanID, true
}

type statusRecorder struct {
	responseWriter http.ResponseWriter
	statusCode     int
	wroteHeader    bool
}

// wrapResponseWriter returns a writer that records status and only exposes
// optional interfaces implemented by the inner writer.
func wrapResponseWriter(response http.ResponseWriter) (http.ResponseWriter, *statusRecorder) {
	recorder := &statusRecorder{responseWriter: response, statusCode: http.StatusOK}
	_, hasFlusher := response.(http.Flusher)
	_, hasHijacker := response.(http.Hijacker)
	_, hasPusher := response.(http.Pusher)

	var wrapped http.ResponseWriter = recorder
	switch {
	case hasFlusher && hasHijacker && hasPusher:
		wrapped = &flushHijackPushRecorder{statusRecorder: recorder}
	case hasFlusher && hasHijacker:
		wrapped = &flushHijackRecorder{statusRecorder: recorder}
	case hasFlusher && hasPusher:
		wrapped = &flushPushRecorder{statusRecorder: recorder}
	case hasHijacker && hasPusher:
		wrapped = &hijackPushRecorder{statusRecorder: recorder}
	case hasFlusher:
		wrapped = &flushRecorder{statusRecorder: recorder}
	case hasHijacker:
		wrapped = &hijackRecorder{statusRecorder: recorder}
	case hasPusher:
		wrapped = &pushRecorder{statusRecorder: recorder}
	}
	return wrapped, recorder
}

// Header returns the wrapped response headers.
func (recorder *statusRecorder) Header() http.Header {
	return recorder.responseWriter.Header()
}

// Write records a 200 status if WriteHeader was not called, then writes the body.
func (recorder *statusRecorder) Write(body []byte) (int, error) {
	if !recorder.wroteHeader {
		recorder.WriteHeader(http.StatusOK)
	}
	return recorder.responseWriter.Write(body)
}

// WriteHeader captures the status code and forwards it once.
func (recorder *statusRecorder) WriteHeader(statusCode int) {
	if recorder.wroteHeader {
		return
	}
	recorder.wroteHeader = true
	recorder.statusCode = statusCode
	recorder.responseWriter.WriteHeader(statusCode)
}

// Unwrap returns the underlying ResponseWriter for http.ResponseController.
func (recorder *statusRecorder) Unwrap() http.ResponseWriter {
	return recorder.responseWriter
}

// flush forwards Flush when the underlying writer supports it.
func (recorder *statusRecorder) flush() {
	if flusher, ok := recorder.responseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// hijack forwards Hijack when the underlying writer supports it.
func (recorder *statusRecorder) hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := recorder.responseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("sseq: ResponseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

// push forwards HTTP/2 server push when the underlying writer supports it.
func (recorder *statusRecorder) push(target string, options *http.PushOptions) error {
	pusher, ok := recorder.responseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, options)
}

type flushRecorder struct{ *statusRecorder }

// Flush implements http.Flusher.
func (recorder *flushRecorder) Flush() { recorder.flush() }

type hijackRecorder struct{ *statusRecorder }

// Hijack implements http.Hijacker.
func (recorder *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return recorder.hijack()
}

type pushRecorder struct{ *statusRecorder }

// Push implements http.Pusher.
func (recorder *pushRecorder) Push(target string, options *http.PushOptions) error {
	return recorder.push(target, options)
}

type flushHijackRecorder struct{ *statusRecorder }

// Flush implements http.Flusher.
func (recorder *flushHijackRecorder) Flush() { recorder.flush() }

// Hijack implements http.Hijacker.
func (recorder *flushHijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return recorder.hijack()
}

type flushPushRecorder struct{ *statusRecorder }

// Flush implements http.Flusher.
func (recorder *flushPushRecorder) Flush() { recorder.flush() }

// Push implements http.Pusher.
func (recorder *flushPushRecorder) Push(target string, options *http.PushOptions) error {
	return recorder.push(target, options)
}

type hijackPushRecorder struct{ *statusRecorder }

// Hijack implements http.Hijacker.
func (recorder *hijackPushRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return recorder.hijack()
}

// Push implements http.Pusher.
func (recorder *hijackPushRecorder) Push(target string, options *http.PushOptions) error {
	return recorder.push(target, options)
}

type flushHijackPushRecorder struct{ *statusRecorder }

// Flush implements http.Flusher.
func (recorder *flushHijackPushRecorder) Flush() { recorder.flush() }

// Hijack implements http.Hijacker.
func (recorder *flushHijackPushRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return recorder.hijack()
}

// Push implements http.Pusher.
func (recorder *flushHijackPushRecorder) Push(target string, options *http.PushOptions) error {
	return recorder.push(target, options)
}

// isHex reports whether value is lowercase hexadecimal of the given length.
func isHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

// isAllZero reports whether value contains only the character '0'.
func isAllZero(value string) bool {
	for _, character := range value {
		if character != '0' {
			return false
		}
	}
	return len(value) > 0
}
