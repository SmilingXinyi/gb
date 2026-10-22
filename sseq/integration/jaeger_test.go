package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
)

const (
	jaegerQueryEndpoint = "http://127.0.0.1:16686"
	jaegerOTLPEndpoint  = "http://127.0.0.1:4318/v1/traces"
	jaegerPollTimeout   = 8 * time.Second
	jaegerPollInterval  = 200 * time.Millisecond
	jaegerParentName    = "checkout"
	jaegerChildName     = "charge"
	jaegerServiceName   = "sseq-jaeger"
)

// TestIntegrationOTLPJaeger exports a parent span and a failing child span to Jaeger,
// then reads them back from /api/v3/traces/{traceId}.
func TestIntegrationOTLPJaeger(t *testing.T) {
	if os.Getenv("SSEQ_SKIP_INTEGRATION") == "1" {
		t.Skip("integration disabled by SSEQ_SKIP_INTEGRATION=1")
	}
	queryEndpoint := jaegerEndpoint("SSEQ_JAEGER_QUERY_URL", jaegerQueryEndpoint)
	otlpEndpoint := jaegerEndpoint("SSEQ_JAEGER_OTLP_URL", jaegerOTLPEndpoint)
	if !jaegerAvailable(queryEndpoint) {
		t.Skip("Jaeger is not available")
	}

	exportErrors := make(chan error, 4)
	if err := sseq.SetupOTLP(otlpEndpoint, jaegerServiceName,
		sseq.WithResourceAttributes(map[string]any{"deployment.environment": "test"}),
		sseq.WithBatchSize(10),
		sseq.WithFlushInterval(time.Hour),
		sseq.WithErrorHandler(func(err error) {
			select {
			case exportErrors <- err:
			default:
			}
			t.Errorf("export: %v", err)
		}),
	); err != nil {
		t.Fatalf("SetupOTLP() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })

	parentCtx, endParent := sseq.Start(context.Background(), jaegerParentName, "server")
	traceID, parentSpanID, ok := sseq.IDs(parentCtx)
	if !ok {
		endParent()
		t.Fatal("missing trace id on parent span")
	}
	sseq.Set(parentCtx, "http.route", "/checkout")
	sseq.Set(parentCtx, "cart.items", []any{"book", "pen"})
	sseq.Event(parentCtx, "cart.loaded", "item.count", 2)

	childErr := sseq.Trace(parentCtx, jaegerChildName, "client", func(ctx context.Context) error {
		sseq.Set(ctx, "payment.amount", 19.5)
		sseq.Set(ctx, "payment.approved", false)
		time.Sleep(40 * time.Millisecond)
		return errors.New("card declined")
	})
	endParent()
	if childErr == nil || childErr.Error() != "card declined" {
		t.Fatalf("child error = %v", childErr)
	}
	if err := sseq.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	select {
	case err := <-exportErrors:
		t.Fatalf("export error = %v", err)
	default:
	}

	spans, err := queryJaegerSpans(queryEndpoint, traceID)
	if err != nil {
		t.Fatalf("query jaeger: %v", err)
	}
	parent, child := requireParentAndChild(t, spans, traceID, parentSpanID)

	if parent.ScopeName != "sseq" || child.ScopeName != "sseq" {
		t.Fatalf("scope names = (%q, %q)", parent.ScopeName, child.ScopeName)
	}
	if parent.ServiceName != jaegerServiceName || child.ServiceName != jaegerServiceName {
		t.Fatalf("service names = (%q, %q)", parent.ServiceName, child.ServiceName)
	}
	if parent.ResourceAttributes["deployment.environment"] != "test" {
		t.Fatalf("resource = %#v", parent.ResourceAttributes)
	}
	if !kindMatches(parent.Kind, "server", 2) || !kindMatches(child.Kind, "client", 3) {
		t.Fatalf("kinds = (%#v, %#v)", parent.Kind, child.Kind)
	}
	if isErrorCode(parent.StatusCode) {
		t.Fatalf("parent status = %#v %q", parent.StatusCode, parent.StatusMessage)
	}
	if !isErrorCode(child.StatusCode) || child.StatusMessage != "card declined" {
		t.Fatalf("child status = %#v %q", child.StatusCode, child.StatusMessage)
	}

	requireAttribute(t, parent, "http.route", "/checkout")
	requireAttribute(t, parent, "cart.items", []any{"book", "pen"})
	requireAttribute(t, child, "payment.amount", 19.5)
	requireAttribute(t, child, "payment.approved", false)

	loaded, found := eventByName(parent, "cart.loaded")
	if !found || loaded.Attributes["item.count"] != int64(2) {
		t.Fatalf("cart.loaded = %#v", loaded)
	}
	exception, found := eventByName(child, "exception")
	if !found || exception.Attributes["exception.message"] != "card declined" {
		t.Fatalf("exception = %#v", exception)
	}
	if exception.Attributes["exception.type"] != "*errors.errorString" {
		t.Fatalf("exception type = %#v", exception.Attributes["exception.type"])
	}

	if child.Duration < 20*time.Millisecond {
		t.Fatalf("child duration = %s", child.Duration)
	}
	if child.Start.Before(parent.Start) || child.End.After(parent.End) {
		t.Fatalf("child [%s, %s] is outside parent [%s, %s]", child.Start, child.End, parent.Start, parent.End)
	}
	if parent.Duration < child.Duration {
		t.Fatalf("parent duration %s is shorter than child %s", parent.Duration, child.Duration)
	}
}

type queriedEvent struct {
	Name       string
	Time       time.Time
	Attributes map[string]any
}

type queriedSpan struct {
	Name               string
	TraceID            string
	SpanID             string
	ParentSpanID       string
	Kind               any
	Start              time.Time
	End                time.Time
	Duration           time.Duration
	StatusCode         any
	StatusMessage      string
	Attributes         map[string]any
	Events             []queriedEvent
	ServiceName        string
	ResourceAttributes map[string]any
	ScopeName          string
}

// jaegerEndpoint returns an override from the environment, or fallback.
func jaegerEndpoint(envName, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value
	}
	return fallback
}

// jaegerAvailable reports whether the Jaeger query UI responds.
func jaegerAvailable(queryEndpoint string) bool {
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get(strings.TrimRight(queryEndpoint, "/") + "/")
	if err != nil {
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode == http.StatusOK
}

// queryJaegerSpans polls until the trace is visible. Jaeger answers 404 until its batch is flushed.
func queryJaegerSpans(queryEndpoint, traceID string) ([]queriedSpan, error) {
	queryURL := strings.TrimRight(queryEndpoint, "/") + "/api/v3/traces/" + url.PathEscape(traceID)
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(jaegerPollTimeout)
	var lastErr error
	for {
		spans, err := fetchJaegerSpans(client, queryURL)
		if err == nil && len(spans) > 0 {
			return spans, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("trace %s not found", traceID)
		}
		if !time.Now().Before(deadline) {
			return nil, lastErr
		}
		time.Sleep(jaegerPollInterval)
	}
}

// fetchJaegerSpans reads one query response. A 404 is not an error so the caller can retry.
func fetchJaegerSpans(client *http.Client, queryURL string) ([]queriedSpan, error) {
	request, err := http.NewRequest(http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", response.StatusCode, truncateText(body))
	}
	spans, err := parseJaegerSpans(body)
	if err != nil {
		return nil, err
	}
	return spans, nil
}

// parseJaegerSpans reads the OTLP JSON document returned by Jaeger API v3.
func parseJaegerSpans(body []byte) ([]queriedSpan, error) {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode jaeger trace: %w: %s", err, truncateText(body))
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("jaeger payload = %T", decoded)
	}
	if result, exists := root["result"]; exists {
		resultMap, isMap := result.(map[string]any)
		if !isMap {
			return nil, fmt.Errorf("jaeger result = %T", result)
		}
		root = resultMap
	}
	resourceSpans, _ := root["resourceSpans"].([]any)
	spans := make([]queriedSpan, 0, len(resourceSpans))
	for _, rawResourceSpan := range resourceSpans {
		resourceSpan, _ := rawResourceSpan.(map[string]any)
		resource, _ := resourceSpan["resource"].(map[string]any)
		resourceAttributes := normalizeAttributes(resource["attributes"])
		serviceName, _ := resourceAttributes["service.name"].(string)
		scopeSpans, _ := resourceSpan["scopeSpans"].([]any)
		for _, rawScopeSpan := range scopeSpans {
			scopeSpan, _ := rawScopeSpan.(map[string]any)
			scope, _ := scopeSpan["scope"].(map[string]any)
			scopeName, _ := scope["name"].(string)
			rawSpans, _ := scopeSpan["spans"].([]any)
			for _, rawSpan := range rawSpans {
				spanObject, _ := rawSpan.(map[string]any)
				spans = append(spans, queriedSpanFromMap(spanObject, serviceName, scopeName, resourceAttributes))
			}
		}
	}
	return spans, nil
}

// queriedSpanFromMap copies one OTLP span into the assertion view.
func queriedSpanFromMap(span map[string]any, serviceName, scopeName string, resourceAttributes map[string]any) queriedSpan {
	start := parseUnixNano(span["startTimeUnixNano"])
	end := parseUnixNano(span["endTimeUnixNano"])
	var statusCode any
	var statusMessage string
	if status, ok := span["status"].(map[string]any); ok {
		statusCode = status["code"]
		statusMessage, _ = status["message"].(string)
	}
	parentSpanID, _ := span["parentSpanId"].(string)
	traceID, _ := span["traceId"].(string)
	spanID, _ := span["spanId"].(string)
	name, _ := span["name"].(string)
	return queriedSpan{
		Name:               name,
		TraceID:            traceID,
		SpanID:             spanID,
		ParentSpanID:       parentSpanID,
		Kind:               span["kind"],
		Start:              start,
		End:                end,
		Duration:           end.Sub(start),
		StatusCode:         statusCode,
		StatusMessage:      statusMessage,
		Attributes:         normalizeAttributes(span["attributes"]),
		Events:             normalizeEvents(span["events"]),
		ServiceName:        serviceName,
		ResourceAttributes: resourceAttributes,
		ScopeName:          scopeName,
	}
}

// normalizeAttributes indexes OTLP key/value attributes by key.
func normalizeAttributes(raw any) map[string]any {
	items, _ := raw.([]any)
	if len(items) == 0 {
		return nil
	}
	attributes := make(map[string]any, len(items))
	for _, item := range items {
		attribute, _ := item.(map[string]any)
		key, _ := attribute["key"].(string)
		if key == "" {
			continue
		}
		attributes[key] = normalizeAnyValue(attribute["value"])
	}
	return attributes
}

// normalizeEvents copies OTLP events into the assertion view.
func normalizeEvents(raw any) []queriedEvent {
	items, _ := raw.([]any)
	events := make([]queriedEvent, 0, len(items))
	for _, item := range items {
		event, _ := item.(map[string]any)
		name, _ := event["name"].(string)
		events = append(events, queriedEvent{
			Name:       name,
			Time:       parseUnixNano(event["timeUnixNano"]),
			Attributes: normalizeAttributes(event["attributes"]),
		})
	}
	return events
}

// normalizeAnyValue converts an OTLP AnyValue into a Go value.
func normalizeAnyValue(value any) any {
	object, ok := value.(map[string]any)
	if !ok {
		return value
	}
	if text, exists := object["stringValue"]; exists {
		return text
	}
	if boolean, exists := object["boolValue"]; exists {
		return boolean
	}
	if number, exists := object["doubleValue"]; exists {
		return number
	}
	if raw, exists := object["intValue"]; exists {
		return parseFlexibleInt(raw)
	}
	if array, exists := object["arrayValue"].(map[string]any); exists {
		items, _ := array["values"].([]any)
		decoded := make([]any, 0, len(items))
		for _, item := range items {
			decoded = append(decoded, normalizeAnyValue(item))
		}
		return decoded
	}
	if len(object) == 0 {
		return nil
	}
	return object
}

// parseFlexibleInt reads an intValue that Jaeger may return as a string or a number.
func parseFlexibleInt(value any) any {
	switch typed := value.(type) {
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return typed
		}
		return parsed
	case float64:
		return int64(typed)
	default:
		return value
	}
}

// parseUnixNano reads an OTLP timestamp string.
func parseUnixNano(value any) time.Time {
	text, ok := value.(string)
	if !ok || text == "" {
		return time.Time{}
	}
	nanos, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(0, nanos).UTC()
}

// requireParentAndChild finds the two exported spans and checks their link.
func requireParentAndChild(t *testing.T, spans []queriedSpan, traceID, parentSpanID string) (queriedSpan, queriedSpan) {
	t.Helper()
	parent, parentFound := spanByName(spans, jaegerParentName)
	child, childFound := spanByName(spans, jaegerChildName)
	if !parentFound || !childFound {
		t.Fatalf("spans = %+v", spans)
	}
	if !strings.EqualFold(parent.TraceID, traceID) || !strings.EqualFold(child.TraceID, traceID) {
		t.Fatalf("trace ids = (%s, %s), want %s", parent.TraceID, child.TraceID, traceID)
	}
	if len(parent.SpanID) != 16 || len(child.SpanID) != 16 || len(parent.TraceID) != 32 {
		t.Fatalf("id lengths parent=%s child=%s trace=%s", parent.SpanID, child.SpanID, parent.TraceID)
	}
	if parent.ParentSpanID != "" {
		t.Fatalf("parent parentSpanId = %q", parent.ParentSpanID)
	}
	if !strings.EqualFold(parent.SpanID, parentSpanID) {
		t.Fatalf("parent span id = %s, want %s", parent.SpanID, parentSpanID)
	}
	if !strings.EqualFold(child.ParentSpanID, parent.SpanID) {
		t.Fatalf("child parentSpanId = %s, want %s", child.ParentSpanID, parent.SpanID)
	}
	return parent, child
}

// spanByName returns the span with the given name.
func spanByName(spans []queriedSpan, name string) (queriedSpan, bool) {
	for _, span := range spans {
		if span.Name == name {
			return span, true
		}
	}
	return queriedSpan{}, false
}

// eventByName returns the event with the given name.
func eventByName(span queriedSpan, name string) (queriedEvent, bool) {
	for _, event := range span.Events {
		if event.Name == name {
			return event, true
		}
	}
	return queriedEvent{}, false
}

// requireAttribute fails when key is missing or differs from want.
func requireAttribute(t *testing.T, span queriedSpan, key string, want any) {
	t.Helper()
	got, ok := span.Attributes[key]
	if !ok {
		t.Fatalf("span %q missing %q in %#v", span.Name, key, span.Attributes)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("span %q attribute %q = %#v, want %#v", span.Name, key, got, want)
	}
}

// isErrorCode reports whether an OTLP status code means ERROR.
func isErrorCode(value any) bool {
	switch typed := value.(type) {
	case float64:
		return typed == 2
	case string:
		upper := strings.ToUpper(typed)
		return upper == "2" || upper == "STATUS_CODE_ERROR"
	default:
		return false
	}
}

// kindMatches accepts the numeric OTLP kind or its proto enum name.
func kindMatches(value any, name string, number float64) bool {
	switch typed := value.(type) {
	case float64:
		return typed == number
	case string:
		upper := strings.ToUpper(typed)
		return upper == strings.ToUpper(name) || upper == "SPAN_KIND_"+strings.ToUpper(name)
	default:
		return false
	}
}

// truncateText returns the start of a response body for an error message.
func truncateText(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) > 512 {
		return text[:512]
	}
	return text
}
