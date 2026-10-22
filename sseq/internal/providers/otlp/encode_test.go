package otlp

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

type userID int

type sampleBox struct {
	Name string
}

func TestEncodeAttributeTypes(t *testing.T) {
	start := time.Unix(10, 0).UTC()
	end := time.Unix(10, 250).UTC()
	cycle := make([]any, 1)
	cycle[0] = cycle

	payload, err := NewEncoder("api", nil).Encode(ss.SpanEvent{
		Name:      "typed",
		TraceID:   "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		SpanID:    "BBBBBBBBBBBBBBBB",
		StartTime: start,
		EndTime:   end,
		Attributes: map[string]any{
			"string.attr":      "hello",
			"bool.attr":        false,
			"zero.attr":        0,
			"int.attr":         7,
			"int64.attr":       int64(math.MaxInt64),
			"uint.attr":        uint32(9),
			"uint64.attr":      uint64(math.MaxUint64),
			"named.attr":       userID(8),
			"float.attr":       1.5,
			"nan.attr":         math.NaN(),
			"inf.attr":         math.Inf(-1),
			"array.attr":       []any{"a", int64(2), true},
			"slice.attr":       []int{3, 4},
			"bytes.attr":       []byte("hi"),
			"number.attr":      json.Number("11"),
			"floatnumber.attr": json.Number("1.25"),
			"nil.attr":         nil,
			"struct.attr":      sampleBox{Name: "box"},
			"chan.attr":        make(chan int),
			"map.attr":         map[string]int{"n": 1},
			"mixed.attr":       []any{map[string]any{"a": 1}, make(chan int), "ok"},
			"cycle.attr":       cycle,
		},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if bytes.Contains(payload, []byte("\n")) {
		t.Fatal("encoded span must stay on one line so a batch can be merged")
	}

	span := decodeSpan(t, payload)
	if span["traceId"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || span["spanId"] != "bbbbbbbbbbbbbbbb" {
		t.Fatalf("ids = (%v, %v)", span["traceId"], span["spanId"])
	}
	if _, exists := span["parentSpanId"]; exists {
		t.Fatalf("root parentSpanId = %v", span["parentSpanId"])
	}
	if span["startTimeUnixNano"] != strconv.FormatInt(start.UnixNano(), 10) {
		t.Fatalf("startTimeUnixNano = %#v", span["startTimeUnixNano"])
	}
	if _, ok := span["startTimeUnixNano"].(string); !ok {
		t.Fatalf("startTimeUnixNano type = %T", span["startTimeUnixNano"])
	}
	if span["endTimeUnixNano"] != strconv.FormatInt(end.UnixNano(), 10) {
		t.Fatalf("endTimeUnixNano = %#v", span["endTimeUnixNano"])
	}

	assertStringValue(t, span, "string.attr", "hello")
	assertBoolValue(t, span, "bool.attr", false)
	assertIntValue(t, span, "zero.attr", "0")
	assertIntValue(t, span, "int.attr", "7")
	assertIntValue(t, span, "int64.attr", strconv.FormatInt(math.MaxInt64, 10))
	assertIntValue(t, span, "uint.attr", "9")
	assertStringValue(t, span, "uint64.attr", strconv.FormatUint(math.MaxUint64, 10))
	assertIntValue(t, span, "named.attr", "8")
	assertDoubleValue(t, span, "float.attr", 1.5)
	assertStringValue(t, span, "nan.attr", "NaN")
	assertStringValue(t, span, "inf.attr", "-Inf")
	assertStringValue(t, span, "bytes.attr", "hi")
	assertIntValue(t, span, "number.attr", "11")
	assertDoubleValue(t, span, "floatnumber.attr", 1.25)
	assertStringValue(t, span, "struct.attr", "{box}")
	if value := attributeValue(t, span, "nil.attr"); len(value) != 0 {
		t.Fatalf("nil attribute = %#v", value)
	}
	if _, ok := attributeValue(t, span, "chan.attr")["stringValue"].(string); !ok {
		t.Fatalf("chan attribute = %#v", attributeValue(t, span, "chan.attr"))
	}
	mapText, _ := attributeValue(t, span, "map.attr")["stringValue"].(string)
	if mapText == "" {
		t.Fatalf("map attribute = %#v", attributeValue(t, span, "map.attr"))
	}

	array := attributeValue(t, span, "array.attr")["arrayValue"].(map[string]any)
	values := array["values"].([]any)
	if len(values) != 3 {
		t.Fatalf("array = %#v", values)
	}
	if values[0].(map[string]any)["stringValue"] != "a" || values[1].(map[string]any)["intValue"] != "2" || values[2].(map[string]any)["boolValue"] != true {
		t.Fatalf("array values = %#v", values)
	}

	intSlice := attributeValue(t, span, "slice.attr")["arrayValue"].(map[string]any)["values"].([]any)
	if intSlice[0].(map[string]any)["intValue"] != "3" || intSlice[1].(map[string]any)["intValue"] != "4" {
		t.Fatalf("slice = %#v", intSlice)
	}

	mixed := attributeValue(t, span, "mixed.attr")["arrayValue"].(map[string]any)["values"].([]any)
	if len(mixed) != 3 || mixed[2].(map[string]any)["stringValue"] != "ok" {
		t.Fatalf("mixed = %#v", mixed)
	}
	if _, ok := mixed[0].(map[string]any)["stringValue"].(string); !ok {
		t.Fatalf("mixed map element = %#v", mixed[0])
	}
	if !bytes.Contains(payload, []byte(`"<cycle>"`)) {
		t.Fatalf("cycle was not degraded: %s", payload)
	}
	if span["name"] != "typed" {
		t.Fatalf("name = %v", span["name"])
	}
}

func TestEncodeSpanKinds(t *testing.T) {
	kinds := []struct {
		name string
		want float64
	}{
		{name: "server", want: 2},
		{name: "Server", want: 2},
		{name: "client", want: 3},
		{name: "CLIENT", want: 3},
		{name: "producer", want: 4},
		{name: "consumer", want: 5},
		{name: "internal", want: 1},
		{name: "", want: 1},
		{name: "database", want: 1},
	}
	for _, test := range kinds {
		payload, err := Encoder{}.Encode(ss.SpanEvent{
			Name:      test.name,
			TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			SpanID:    "bbbbbbbbbbbbbbbb",
			SpanKind:  test.name,
			StartTime: time.Unix(1, 0).UTC(),
			EndTime:   time.Unix(2, 0).UTC(),
		})
		if err != nil {
			t.Fatalf("Encode(%q) error = %v", test.name, err)
		}
		span := decodeSpan(t, payload)
		if span["kind"] != test.want {
			t.Fatalf("kind %q = %#v, want %v", test.name, span["kind"], test.want)
		}
	}
}

func TestEncodeChildParentAndError(t *testing.T) {
	eventTime := time.Unix(4, 0).UTC()
	end := time.Unix(5, 0).UTC()
	payload, err := Encoder{}.Encode(ss.SpanEvent{
		Name:          "charge",
		TraceID:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:        "cccccccccccccccc",
		ParentID:      "BBBBBBBBBBBBBBBB",
		SpanKind:      "client",
		StartTime:     time.Unix(3, 0).UTC(),
		EndTime:       end,
		HasError:      true,
		StatusMessage: "card declined",
		Attributes: map[string]any{
			"exception.type":    "*errors.errorString",
			"exception.message": "card declined",
			"user.id":           "42",
			"broken":            make(chan struct{}),
		},
		Events: []ss.TimedEvent{{
			Name:       "checkpoint",
			Time:       eventTime,
			Attributes: map[string]any{"step": "auth", "bad": make(chan int)},
		}},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	span := decodeSpan(t, payload)
	if span["parentSpanId"] != "bbbbbbbbbbbbbbbb" {
		t.Fatalf("parentSpanId = %v", span["parentSpanId"])
	}
	status, ok := span["status"].(map[string]any)
	if !ok || status["code"] != float64(2) || status["message"] != "card declined" {
		t.Fatalf("status = %#v", span["status"])
	}
	assertStringValue(t, span, "user.id", "42")
	if _, isString := attributeValue(t, span, "broken")["stringValue"].(string); !isString {
		t.Fatalf("broken attribute = %#v", attributeValue(t, span, "broken"))
	}

	events, ok := span["events"].([]any)
	if !ok || len(events) != 2 {
		t.Fatalf("events = %#v", span["events"])
	}
	checkpoint := events[0].(map[string]any)
	if checkpoint["name"] != "checkpoint" || checkpoint["timeUnixNano"] != strconv.FormatInt(eventTime.UnixNano(), 10) {
		t.Fatalf("checkpoint = %#v", checkpoint)
	}
	exception := events[1].(map[string]any)
	if exception["name"] != "exception" || exception["timeUnixNano"] != strconv.FormatInt(end.UnixNano(), 10) {
		t.Fatalf("exception = %#v", exception)
	}
	exceptionAttributes := attributesByKey(t, exception["attributes"])
	if exceptionAttributes["exception.message"].(map[string]any)["stringValue"] != "card declined" {
		t.Fatalf("exception attributes = %#v", exceptionAttributes)
	}
	if exceptionAttributes["exception.type"].(map[string]any)["stringValue"] != "*errors.errorString" {
		t.Fatalf("exception type = %#v", exceptionAttributes["exception.type"])
	}
}

func TestEncodeHTTPErrorUsesStatusMessage(t *testing.T) {
	payload, err := Encoder{}.Encode(ss.SpanEvent{
		Name:          "GET",
		TraceID:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:        "bbbbbbbbbbbbbbbb",
		StartTime:     time.Unix(1, 0).UTC(),
		EndTime:       time.Unix(2, 0).UTC(),
		HasError:      true,
		StatusMessage: "HTTP 500",
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	span := decodeSpan(t, payload)
	status := span["status"].(map[string]any)
	if status["code"] != float64(2) || status["message"] != "HTTP 500" {
		t.Fatalf("status = %#v", status)
	}
	events := span["events"].([]any)
	exception := events[0].(map[string]any)
	attributes := attributesByKey(t, exception["attributes"])
	if exception["name"] != "exception" || attributes["exception.message"].(map[string]any)["stringValue"] != "HTTP 500" {
		t.Fatalf("exception = %#v", exception)
	}
}

func TestEncodeResourceAttributes(t *testing.T) {
	payload, err := NewEncoder("api", map[string]any{
		"service.name":           "other",
		"deployment.environment": "prod",
		"instance":               3,
	}).Encode(ss.SpanEvent{
		Name:      "root",
		TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:    "bbbbbbbbbbbbbbbb",
		StartTime: time.Unix(1, 0).UTC(),
		EndTime:   time.Unix(2, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	resource, scope := decodeResourceScope(t, payload)
	attributes := attributesByKey(t, resource["attributes"])
	if attributes["service.name"].(map[string]any)["stringValue"] != "api" {
		t.Fatalf("service.name = %#v", attributes["service.name"])
	}
	if attributes["deployment.environment"].(map[string]any)["stringValue"] != "prod" {
		t.Fatalf("environment = %#v", attributes["deployment.environment"])
	}
	if attributes["instance"].(map[string]any)["intValue"] != "3" {
		t.Fatalf("instance = %#v", attributes["instance"])
	}
	if scope["name"] != scopeName {
		t.Fatalf("scope = %#v", scope)
	}

	fallback, err := NewEncoder("", map[string]any{"service.name": "from-map"}).Encode(ss.SpanEvent{
		Name:      "root",
		TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:    "bbbbbbbbbbbbbbbb",
		StartTime: time.Unix(1, 0).UTC(),
		EndTime:   time.Unix(2, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	fallbackResource, _ := decodeResourceScope(t, fallback)
	fallbackAttributes := attributesByKey(t, fallbackResource["attributes"])
	if fallbackAttributes["service.name"].(map[string]any)["stringValue"] != "from-map" {
		t.Fatalf("fallback service.name = %#v", fallbackAttributes["service.name"])
	}
}

func decodeSpan(t *testing.T, payload []byte) map[string]any {
	t.Helper()
	_, _, spans := decodeDocument(t, payload)
	if len(spans) != 1 {
		t.Fatalf("spans = %#v", spans)
	}
	return spans[0]
}

func decodeResourceScope(t *testing.T, payload []byte) (map[string]any, map[string]any) {
	t.Helper()
	resource, scope, _ := decodeDocument(t, payload)
	return resource, scope
}

func decodeDocument(t *testing.T, payload []byte) (map[string]any, map[string]any, []map[string]any) {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	resourceSpans, ok := document["resourceSpans"].([]any)
	if !ok || len(resourceSpans) != 1 {
		t.Fatalf("resourceSpans = %#v", document["resourceSpans"])
	}
	resourceSpan := resourceSpans[0].(map[string]any)
	resource, _ := resourceSpan["resource"].(map[string]any)
	scopeSpans := resourceSpan["scopeSpans"].([]any)
	scopeSpan := scopeSpans[0].(map[string]any)
	scope, _ := scopeSpan["scope"].(map[string]any)
	rawSpans := scopeSpan["spans"].([]any)
	spans := make([]map[string]any, 0, len(rawSpans))
	for _, rawSpan := range rawSpans {
		spans = append(spans, rawSpan.(map[string]any))
	}
	return resource, scope, spans
}

func attributeValue(t *testing.T, span map[string]any, key string) map[string]any {
	t.Helper()
	attributes := attributesByKey(t, span["attributes"])
	value, ok := attributes[key].(map[string]any)
	if !ok {
		t.Fatalf("attribute %q = %#v", key, attributes[key])
	}
	return value
}

func attributesByKey(t *testing.T, raw any) map[string]any {
	t.Helper()
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("attributes = %#v", raw)
	}
	indexed := make(map[string]any, len(items))
	for _, item := range items {
		attribute := item.(map[string]any)
		indexed[attribute["key"].(string)] = attribute["value"]
	}
	return indexed
}

func assertStringValue(t *testing.T, span map[string]any, key string, want string) {
	t.Helper()
	if got := attributeValue(t, span, key)["stringValue"]; got != want {
		t.Fatalf("%s = %#v, want %q", key, got, want)
	}
}

func assertBoolValue(t *testing.T, span map[string]any, key string, want bool) {
	t.Helper()
	if got := attributeValue(t, span, key)["boolValue"]; got != want {
		t.Fatalf("%s = %#v, want %v", key, got, want)
	}
}

func assertIntValue(t *testing.T, span map[string]any, key string, want string) {
	t.Helper()
	if got := attributeValue(t, span, key)["intValue"]; got != want {
		t.Fatalf("%s = %#v, want %q", key, got, want)
	}
}

func assertDoubleValue(t *testing.T, span map[string]any, key string, want float64) {
	t.Helper()
	if got := attributeValue(t, span, key)["doubleValue"]; got != want {
		t.Fatalf("%s = %#v, want %v", key, got, want)
	}
}
