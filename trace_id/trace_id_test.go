package trace_id

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	id, err := New()
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestNewString(t *testing.T) {
	idString, err := NewString()
	assert.NoError(t, err)
	assert.NotEmpty(t, idString)
	assert.Contains(t, idString, "-")

	id, err := uuid.Parse(idString)
	assert.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestNewHex(t *testing.T) {
	idHex, err := NewHex()
	assert.NoError(t, err)
	assert.Len(t, idHex, 32)
	assert.NotContains(t, idHex, "-")
	assert.Equal(t, idHex, strings.ToLower(idHex))
}

func TestMustNew(t *testing.T) {
	assert.NotPanics(t, func() {
		id := MustNew()
		assert.NotEqual(t, uuid.Nil, id)
		assert.Equal(t, uuid.Version(7), id.Version())
	})
}

func TestMustNewString(t *testing.T) {
	assert.NotPanics(t, func() {
		idString := MustNewString()
		assert.NotEmpty(t, idString)
	})
}

func TestMustNewHex(t *testing.T) {
	assert.NotPanics(t, func() {
		idHex := MustNewHex()
		assert.Len(t, idHex, 32)
		assert.NotContains(t, idHex, "-")
	})
}
