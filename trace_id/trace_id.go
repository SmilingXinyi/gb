package trace_id

import (
	"github.com/google/uuid"
)

// New generates a new time-ordered UUID v7 trace ID.
// UUID v7 embeds a timestamp component, making it suitable for database keys and distributed tracing.
func New() (ID, error) {
	generated, err := uuid.NewV7()
	if err != nil {
		return ID{}, err
	}
	return ID(generated), nil
}

// NewString generates a new trace ID and returns its canonical UUID string representation.
func NewString() (string, error) {
	generated, err := New()
	if err != nil {
		return "", err
	}
	return generated.String(), nil
}

// NewHex generates a new trace ID and returns a 32-character lowercase hexadecimal string.
func NewHex() (string, error) {
	generated, err := New()
	if err != nil {
		return "", err
	}
	return generated.Hex(), nil
}

// MustNew generates a new trace ID and panics when generation fails.
// Use New when a fallback strategy is required, for example in request hot paths.
func MustNew() ID {
	generated, err := New()
	if err != nil {
		panic(err)
	}
	return generated
}

// MustNewString generates a new trace ID string and panics when generation fails.
func MustNewString() string {
	return MustNew().String()
}

// MustNewHex generates a new hexadecimal trace ID and panics when generation fails.
func MustNewHex() string {
	return MustNew().Hex()
}
