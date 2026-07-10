package trace_id

import (
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	id, err := New()
	require.NoError(t, err)
	assert.False(t, id.IsZero())
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestNewString(t *testing.T) {
	idString, err := NewString()
	require.NoError(t, err)
	assert.NotEmpty(t, idString)

	parsed, err := uuid.Parse(idString)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), parsed.Version())
}

func TestNewHex(t *testing.T) {
	idHex, err := NewHex()
	require.NoError(t, err)
	assert.Len(t, idHex, 32)
	assert.Equal(t, idHex, strings.ToLower(idHex))
	assert.NotContains(t, idHex, "-")
	assert.True(t, IsValid(idHex))
}

func TestMustNew(t *testing.T) {
	assert.NotPanics(t, func() {
		id := MustNew()
		assert.False(t, id.IsZero())
		assert.Equal(t, uuid.Version(7), id.Version())
	})
}

func TestMustNewString(t *testing.T) {
	assert.NotPanics(t, func() {
		idString := MustNewString()
		assert.NotEmpty(t, idString)
		assert.True(t, IsValid(idString))
	})
}

func TestMustNewHex(t *testing.T) {
	assert.NotPanics(t, func() {
		idHex := MustNewHex()
		assert.Len(t, idHex, 32)
		assert.True(t, IsValid(idHex))
	})
}

func TestNewGeneratesUniqueIDs(t *testing.T) {
	const count = 256
	seen := make(map[string]struct{}, count)

	for index := 0; index < count; index++ {
		idHex, err := NewHex()
		require.NoError(t, err)
		if _, exists := seen[idHex]; exists {
			t.Fatalf("duplicate trace id %q at index %d", idHex, index)
		}
		seen[idHex] = struct{}{}
	}
}

func TestNewGeneratesTimeOrderedIDs(t *testing.T) {
	const count = 64
	previousHex := ""

	for index := 0; index < count; index++ {
		currentHex, err := NewHex()
		require.NoError(t, err)
		if previousHex != "" && currentHex < previousHex {
			t.Fatalf("trace ids out of order: previous=%q current=%q index=%d", previousHex, currentHex, index)
		}
		previousHex = currentHex
	}
}

func TestParseRoundTrip(t *testing.T) {
	generated, err := New()
	require.NoError(t, err)

	uuidCases := []string{
		generated.String(),
		generated.Hex(),
		strings.ToUpper(generated.Hex()),
	}
	for _, raw := range uuidCases {
		parsed, parseErr := Parse(raw)
		require.NoError(t, parseErr, "input=%q", raw)
		assert.Equal(t, generated.Hex(), parsed.Hex())
	}
}

func TestParseRejectsInvalidInput(t *testing.T) {
	invalidInputs := []string{
		"",
		"   ",
		"not-a-trace-id",
		"0123456789abcdef",
		"0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	for _, raw := range invalidInputs {
		_, err := Parse(raw)
		assert.Error(t, err, "input=%q", raw)
		assert.False(t, IsValid(raw), "input=%q", raw)
	}
}

func TestIsValidAcceptsExternalFormats(t *testing.T) {
	assert.True(t, IsValid("0123456789abcdef0123456789abcdef"))
	assert.True(t, IsValid("01234567-89ab-cdef-0123-456789abcdef"))
	assert.False(t, IsValid(""))
}

func TestNewConcurrent(t *testing.T) {
	const goroutines = 32
	const iterations = 128

	var waitGroup sync.WaitGroup
	waitGroup.Add(goroutines)

	seen := make(map[string]struct{}, goroutines*iterations)
	var mutex sync.Mutex

	for worker := 0; worker < goroutines; worker++ {
		go func() {
			defer waitGroup.Done()
			for index := 0; index < iterations; index++ {
				idHex, err := NewHex()
				require.NoError(t, err)

				mutex.Lock()
				if _, exists := seen[idHex]; exists {
					t.Errorf("duplicate trace id %q", idHex)
				}
				seen[idHex] = struct{}{}
				mutex.Unlock()
			}
		}()
	}

	waitGroup.Wait()
}
