package baidu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SmilingXinyi/gb/oss"
)

// TestNewClient_MissingAK 缺少 AK 时 New 应返回 ErrInvalidConfig，不 panic。
func TestNewClient_MissingAK(t *testing.T) {
	_, err := oss.New(oss.ProviderBaidu, oss.Config{SecretKey: "sk"})
	require.Error(t, err)
	var cfgErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &cfgErr)
	assert.Equal(t, "AccessKey", cfgErr.Field)
}

// TestNewClient_MissingSK 缺少 SK 时 New 应返回 ErrInvalidConfig。
func TestNewClient_MissingSK(t *testing.T) {
	_, err := oss.New(oss.ProviderBaidu, oss.Config{AccessKey: "ak"})
	require.Error(t, err)
	var cfgErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &cfgErr)
	assert.Equal(t, "SecretKey", cfgErr.Field)
}

// TestNewClient_UnknownProvider 使用未注册的 provider 应返回 ErrProviderNotSupported。
func TestNewClient_UnknownProvider(t *testing.T) {
	_, err := oss.New("unknown", oss.Config{})
	require.Error(t, err)
	var provErr *oss.ErrProviderNotSupported
	assert.ErrorAs(t, err, &provErr)
}
