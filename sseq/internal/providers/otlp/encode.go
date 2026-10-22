// Package otlp encodes spans as OTLP/HTTP JSON export requests.
//
// Each Encode call returns one single-line JSON document. The HTTP writer merges
// a batch of those documents into one request.
package otlp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

const (
	scopeName = "sseq"

	kindInternal = 1
	kindServer   = 2
	kindClient   = 3
	kindProducer = 4
	kindConsumer = 5

	statusError = 2

	// maxAttributeDepth stops a pathological attribute from recursing forever.
	maxAttributeDepth = 32
)

// Encoder encodes spans as OTLP JSON documents.
type Encoder struct {
	// ResourceAttributes are copied onto every exported resource.
	// service.name is included when SetupOTLP was given a service name.
	ResourceAttributes map[string]any
}

// NewEncoder returns an encoder that attaches service.name and extra resource attributes.
// A non-empty serviceName overwrites a service.name already present in attributes.
func NewEncoder(serviceName string, resourceAttributes map[string]any) Encoder {
	return Encoder{ResourceAttributes: resourceAttributesFor(serviceName, resourceAttributes)}
}

// Encode serializes one span as an OTLP JSON export document.
// An attribute that cannot be represented becomes a string. It does not fail the span.
func (encoder Encoder) Encode(event ss.SpanEvent) ([]byte, error) {
	resource := map[string]any{}
	if attributes := encodeAttributes(encoder.ResourceAttributes); len(attributes) > 0 {
		resource["attributes"] = attributes
	}
	document := map[string]any{
		"resourceSpans": []any{
			map[string]any{
				"resource": resource,
				"scopeSpans": []any{
					map[string]any{
						"scope": map[string]any{"name": scopeName},
						"spans": []any{encodeSpan(event)},
					},
				},
			},
		},
	}
	payload, err := marshalJSON(document)
	if err != nil {
		return nil, fmt.Errorf("marshal otlp span: %w", err)
	}
	return payload, nil
}

// marshalJSON encodes value as one line and leaves <, >, and & readable.
func marshalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buffer.Bytes(), "\n"), nil
}

// resourceAttributesFor copies extra attributes and pins service.name.
func resourceAttributesFor(serviceName string, extra map[string]any) map[string]any {
	attributes := make(map[string]any, len(extra)+1)
	for key, value := range extra {
		if key == "" {
			continue
		}
		attributes[key] = value
	}
	if serviceName != "" {
		attributes["service.name"] = serviceName
	}
	if len(attributes) == 0 {
		return nil
	}
	return attributes
}

// encodeSpan maps one completed span onto an OTLP span object.
func encodeSpan(event ss.SpanEvent) map[string]any {
	span := map[string]any{
		"traceId":           strings.ToLower(event.TraceID),
		"spanId":            strings.ToLower(event.SpanID),
		"name":              event.Name,
		"kind":              spanKind(event.SpanKind),
		"startTimeUnixNano": unixNanoString(event.StartTime),
		"endTimeUnixNano":   unixNanoString(event.EndTime),
	}
	if event.ParentID != "" {
		span["parentSpanId"] = strings.ToLower(event.ParentID)
	}
	if attributes := encodeAttributes(event.Attributes); len(attributes) > 0 {
		span["attributes"] = attributes
	}
	if events := encodeEvents(event); len(events) > 0 {
		span["events"] = events
	}
	if event.HasError {
		status := map[string]any{"code": statusError}
		if event.StatusMessage != "" {
			status["message"] = event.StatusMessage
		}
		span["status"] = status
	}
	return span
}

// encodeEvents maps point events and, for a failed span, appends an exception event.
func encodeEvents(event ss.SpanEvent) []any {
	events := make([]any, 0, len(event.Events)+1)
	for _, timedEvent := range event.Events {
		item := map[string]any{
			"timeUnixNano": unixNanoString(timedEvent.Time),
			"name":         timedEvent.Name,
		}
		if attributes := encodeAttributes(timedEvent.Attributes); len(attributes) > 0 {
			item["attributes"] = attributes
		}
		events = append(events, item)
	}
	if exception, ok := exceptionEvent(event); ok {
		events = append(events, exception)
	}
	return events
}

// exceptionEvent builds the OTel exception event recorded for a failed span.
func exceptionEvent(event ss.SpanEvent) (map[string]any, bool) {
	if !event.HasError {
		return nil, false
	}
	attributes := map[string]any{}
	if value, ok := event.Attributes["exception.type"]; ok {
		attributes["exception.type"] = value
	}
	if value, ok := event.Attributes["exception.message"]; ok {
		attributes["exception.message"] = value
	} else if event.StatusMessage != "" {
		attributes["exception.message"] = event.StatusMessage
	}
	if value, ok := event.Attributes["exception.stacktrace"]; ok {
		attributes["exception.stacktrace"] = value
	}
	eventTime := event.EndTime
	if eventTime.IsZero() {
		eventTime = event.StartTime
	}
	item := map[string]any{
		"timeUnixNano": unixNanoString(eventTime),
		"name":         "exception",
	}
	if encoded := encodeAttributes(attributes); len(encoded) > 0 {
		item["attributes"] = encoded
	}
	return item, true
}

// encodeAttributes converts a map into sorted OTLP key/value pairs.
func encodeAttributes(attributes map[string]any) []any {
	if len(attributes) == 0 {
		return nil
	}
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	encoded := make([]any, 0, len(keys))
	for _, key := range keys {
		encoded = append(encoded, map[string]any{
			"key":   key,
			"value": anyValue(attributes[key]),
		})
	}
	return encoded
}

// anyValue maps a Go value onto an OTLP AnyValue.
// Types that have no AnyValue form become stringValue of fmt.Sprint.
func anyValue(value any) map[string]any {
	return encodeAny(value, map[uintptr]struct{}{}, 0)
}

// encodeAny walks nested slices and remembers their addresses so a cycle cannot loop.
func encodeAny(value any, seen map[uintptr]struct{}, depth int) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if depth > maxAttributeDepth {
		return stringValue(fmt.Sprint(value))
	}
	switch typed := value.(type) {
	case string:
		return stringValue(typed)
	case bool:
		return map[string]any{"boolValue": typed}
	case int:
		return intValue(int64(typed))
	case int8:
		return intValue(int64(typed))
	case int16:
		return intValue(int64(typed))
	case int32:
		return intValue(int64(typed))
	case int64:
		return intValue(typed)
	case uint:
		return uintValue(uint64(typed))
	case uint8:
		return uintValue(uint64(typed))
	case uint16:
		return uintValue(uint64(typed))
	case uint32:
		return uintValue(uint64(typed))
	case uint64:
		return uintValue(typed)
	case float32:
		return floatValue(float64(typed))
	case float64:
		return floatValue(typed)
	case json.Number:
		return jsonNumberValue(typed)
	case []byte:
		return stringValue(string(typed))
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.String:
		return stringValue(reflected.String())
	case reflect.Bool:
		return map[string]any{"boolValue": reflected.Bool()}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intValue(reflected.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintValue(reflected.Uint())
	case reflect.Float32, reflect.Float64:
		return floatValue(reflected.Float())
	case reflect.Slice, reflect.Array:
		return arrayValue(reflected, seen, depth)
	default:
		return stringValue(fmt.Sprint(value))
	}
}

// arrayValue encodes a slice or array as an OTLP ArrayValue.
func arrayValue(reflected reflect.Value, seen map[uintptr]struct{}, depth int) map[string]any {
	if reflected.Kind() == reflect.Slice && !reflected.IsNil() {
		pointer := reflected.Pointer()
		if pointer != 0 {
			if _, exists := seen[pointer]; exists {
				return stringValue("<cycle>")
			}
			seen[pointer] = struct{}{}
			defer delete(seen, pointer)
		}
	}
	values := make([]any, reflected.Len())
	for index := 0; index < reflected.Len(); index++ {
		element := reflected.Index(index)
		if !element.IsValid() || !element.CanInterface() {
			values[index] = stringValue("")
			continue
		}
		values[index] = encodeAny(element.Interface(), seen, depth+1)
	}
	return map[string]any{"arrayValue": map[string]any{"values": values}}
}

// stringValue wraps text as an OTLP string AnyValue.
func stringValue(text string) map[string]any {
	return map[string]any{"stringValue": text}
}

// intValue wraps a signed integer as a decimal string, matching proto JSON.
func intValue(value int64) map[string]any {
	return map[string]any{"intValue": strconv.FormatInt(value, 10)}
}

// uintValue uses intValue when the number fits in int64, otherwise a string.
func uintValue(value uint64) map[string]any {
	if value > math.MaxInt64 {
		return stringValue(strconv.FormatUint(value, 10))
	}
	return map[string]any{"intValue": strconv.FormatUint(value, 10)}
}

// floatValue uses doubleValue for finite numbers and a string for NaN or infinity.
func floatValue(value float64) map[string]any {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return stringValue(nonFiniteFloat(value))
	}
	return map[string]any{"doubleValue": value}
}

// jsonNumberValue keeps whole numbers as intValue and other numbers as doubleValue.
func jsonNumberValue(number json.Number) map[string]any {
	if integer, err := number.Int64(); err == nil {
		return intValue(integer)
	}
	if float, err := number.Float64(); err == nil {
		return floatValue(float)
	}
	return stringValue(number.String())
}

// nonFiniteFloat renders NaN and infinities as text that JSON can carry.
func nonFiniteFloat(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "+Inf"
	default:
		return "-Inf"
	}
}

// spanKind maps sseq kind names onto OTLP span kind numbers.
func spanKind(kind string) int {
	switch strings.ToLower(kind) {
	case "server":
		return kindServer
	case "client":
		return kindClient
	case "producer":
		return kindProducer
	case "consumer":
		return kindConsumer
	default:
		return kindInternal
	}
}

// unixNanoString renders a timestamp as a decimal nanosecond string.
// Zero and pre-epoch times become "0" so the field stays an unsigned quantity.
func unixNanoString(value time.Time) string {
	if value.IsZero() {
		return "0"
	}
	nanos := value.UnixNano()
	if nanos < 0 {
		return "0"
	}
	return strconv.FormatInt(nanos, 10)
}
