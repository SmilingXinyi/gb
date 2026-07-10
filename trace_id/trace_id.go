package trace_id

import (
	"strings"

	"github.com/google/uuid"
)

// New generates a new UUID v7.
func New() (uuid.UUID, error) {
	return uuid.NewV7()
}

// NewString generates a new UUID v7 and returns it as a string with dashes.
func NewString() (string, error) {
	generated, err := New()
	if err != nil {
		return "", err
	}
	return generated.String(), nil
}

// NewHex generates a new UUID v7 and returns a 32-character hexadecimal string without dashes.
func NewHex() (string, error) {
	generated, err := New()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(generated.String(), "-", ""), nil
}

// MustNew generates a new UUID v7 and panics if an error occurs.
func MustNew() uuid.UUID {
	generated, err := New()
	if err != nil {
		panic(err)
	}
	return generated
}

// MustNewString generates a new UUID v7 string with dashes and panics if an error occurs.
func MustNewString() string {
	generated, err := NewString()
	if err != nil {
		panic(err)
	}
	return generated
}

// MustNewHex generates a new UUID v7 hex string without dashes and panics if an error occurs.
func MustNewHex() string {
	generated, err := NewHex()
	if err != nil {
		panic(err)
	}
	return generated
}
