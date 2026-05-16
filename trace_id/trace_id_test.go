package trace_id

import (
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
	idStr, err := NewString()
	assert.NoError(t, err)
	assert.NotEmpty(t, idStr)

	id, err := uuid.Parse(idStr)
	assert.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
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
		idStr := MustNewString()
		assert.NotEmpty(t, idStr)
		id, err := uuid.Parse(idStr)
		assert.NoError(t, err)
		assert.Equal(t, uuid.Version(7), id.Version())
	})
}
