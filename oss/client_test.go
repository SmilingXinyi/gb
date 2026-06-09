package oss

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Register a mock adapter for testing
	mockProvider := Provider("mock")
	Register(mockProvider, func(cfg Config) (Storage, error) {
		return nil, nil // We don't need a real storage for this test
	})

	t.Run("Use Provider type", func(t *testing.T) {
		client, err := New(mockProvider, Config{})
		assert.NoError(t, err)
		assert.Nil(t, client) // Mock returns nil, nil
	})

	t.Run("Use string type", func(t *testing.T) {
		client, err := New("mock", Config{})
		assert.NoError(t, err)
		assert.Nil(t, client)
	})

	t.Run("Unsupported type", func(t *testing.T) {
		_, err := New(123, Config{})
		assert.Error(t, err)
		assert.IsType(t, &ErrProviderNotSupported{}, err)
	})

	t.Run("Unsupported provider string", func(t *testing.T) {
		_, err := New("unknown", Config{})
		assert.Error(t, err)
		assert.IsType(t, &ErrProviderNotSupported{}, err)
		assert.Equal(t, Provider("unknown"), err.(*ErrProviderNotSupported).Provider)
	})
}
