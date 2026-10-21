package ss

import "testing"

func TestSanitizeValueBreaksCycles(t *testing.T) {
	attributes := map[string]any{"user.id": "42"}
	attributes["self"] = attributes

	sanitized, ok := SanitizeValue(attributes).(map[string]any)
	if !ok {
		t.Fatalf("SanitizeValue() = %#v", SanitizeValue(attributes))
	}
	if sanitized["user.id"] != "42" {
		t.Fatalf("user.id = %v", sanitized["user.id"])
	}
	if sanitized["self"] != "<cycle>" {
		t.Fatalf("self = %#v", sanitized["self"])
	}
}
