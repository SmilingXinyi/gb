package tencent_test

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SmilingXinyi/gb/oss"
	_ "github.com/SmilingXinyi/gb/oss/tencent"
)

// TestMain 在所有测试运行前统一加载 .env，避免每个测试重复加载。
func TestMain(m *testing.M) {
	_ = godotenv.Load(".env")
	os.Exit(m.Run())
}

// TestNewClient_MissingAK returns ErrInvalidConfig when AccessKey is empty.
func TestNewClient_MissingAK(t *testing.T) {
	_, err := oss.New(oss.ProviderTencent, oss.Config{SecretKey: "sk", Region: "ap-guangzhou"})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "AccessKey", configErr.Field)
}

// TestNewClient_MissingSK returns ErrInvalidConfig when SecretKey is empty.
func TestNewClient_MissingSK(t *testing.T) {
	_, err := oss.New(oss.ProviderTencent, oss.Config{AccessKey: "ak", Region: "ap-guangzhou"})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "SecretKey", configErr.Field)
}

// TestNewClient_MissingRegionAndEndpoint returns ErrInvalidConfig when both are empty.
func TestNewClient_MissingRegionAndEndpoint(t *testing.T) {
	_, err := oss.New(oss.ProviderTencent, oss.Config{AccessKey: "ak", SecretKey: "sk"})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "Region", configErr.Field)
}

// TestNewClient_WithEndpointOnly succeeds when Endpoint is set without Region.
func TestNewClient_WithEndpointOnly(t *testing.T) {
	client, err := oss.New(oss.ProviderTencent, oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		Endpoint:  "https://example-1250000000.cos.ap-guangzhou.myqcloud.com",
		Bucket:    "example-1250000000",
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// testConfig 读取环境变量，构造 oss.Config。
// 若必填字段缺失则返回 ("", false)，调用方应调用 t.Skip。
func testConfig() (oss.Config, bool) {
	cfg := oss.Config{
		AccessKey: os.Getenv("OSS_TENCENT_ACCESS_KEY"),
		SecretKey: os.Getenv("OSS_TENCENT_SECRET_KEY"),
		Region:    os.Getenv("OSS_TENCENT_REGION"),
		Bucket:    os.Getenv("OSS_TENCENT_BUCKET"),
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return oss.Config{}, false
	}
	return cfg, true
}

// skipIfNoEnv 在未配置凭证时跳过当前测试。
func skipIfNoEnv(t *testing.T) oss.Config {
	t.Helper()
	cfg, ok := testConfig()
	if !ok {
		t.Skip("OSS_TENCENT_ACCESS_KEY / OSS_TENCENT_SECRET_KEY / OSS_TENCENT_BUCKET not set, skipping integration test")
	}
	return cfg
}

// newTestClient 创建测试用 COS 客户端，凭证缺失时自动 skip。
func newTestClient(t *testing.T) (oss.Storage, oss.Config) {
	t.Helper()
	cfg := skipIfNoEnv(t)
	client, err := oss.New(oss.ProviderTencent, cfg)
	if err != nil {
		t.Fatalf("oss.New: %v", err)
	}
	return client, cfg
}
