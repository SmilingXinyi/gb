package trace_id

import (
	"github.com/google/uuid"
)

// New generates a new UUID v7.
// UUID v7 is time-ordered and contains a timestamp component,
// making it suitable for use as a database primary key and for distributed tracing.
func New() (uuid.UUID, error) {
	return uuid.NewV7()
}

// NewString generates a new UUID v7 and returns it as a string.
func NewString() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// MustNew generates a new UUID v7 and panics if an error occurs.
// Suitable for initialization or scenarios where failure is not expected.
func MustNew() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return id
}

// MustNewString generates a new UUID v7 string and panics if an error occurs.
// Suitable for initialization or scenarios where failure is not expected.
func MustNewString() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return id.String()
}
