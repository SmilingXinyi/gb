package cloudflare_test

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SmilingXinyi/gb/oss"
	_ "github.com/SmilingXinyi/gb/oss/cloudflare"
)

// TestMain loads .env once before the external test binary runs.
func TestMain(mainTest *testing.M) {
	_ = godotenv.Load(".env")
	os.Exit(mainTest.Run())
}

// TestNewClient_MissingAccessKey returns ErrInvalidConfig.
func TestNewClient_MissingAccessKey(t *testing.T) {
	_, err := oss.New(oss.ProviderCloudflare, oss.Config{
		SecretKey: "sk",
		AccountID: "abc123def",
	})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "AccessKey", configErr.Field)
}

// TestNewClient_MissingSecretKey returns ErrInvalidConfig.
func TestNewClient_MissingSecretKey(t *testing.T) {
	_, err := oss.New(oss.ProviderCloudflare, oss.Config{
		AccessKey: "ak",
		AccountID: "abc123def",
	})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "SecretKey", configErr.Field)
}

// TestNewClient_MissingAccountAndEndpoint returns ErrInvalidConfig.
func TestNewClient_MissingAccountAndEndpoint(t *testing.T) {
	_, err := oss.New("cloudflare", oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
	})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "AccountID", configErr.Field)
}

// TestNewClient_WithEndpointOnly succeeds without AccountID.
func TestNewClient_WithEndpointOnly(t *testing.T) {
	client, err := oss.New(oss.ProviderCloudflare, oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		Endpoint:  "https://example.r2.cloudflarestorage.com",
		Bucket:    "demo",
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// TestNewClient_WithAccountID succeeds and registers the string provider name.
func TestNewClient_WithAccountID(t *testing.T) {
	client, err := oss.New("cloudflare", oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		AccountID: "abc123def",
		Region:    "auto",
		Bucket:    "demo",
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// testConfig reads Cloudflare credentials from the environment.
func testConfig() (oss.Config, bool) {
	config := oss.Config{
		AccessKey: os.Getenv("OSS_CLOUDFLARE_ACCESS_KEY"),
		SecretKey: os.Getenv("OSS_CLOUDFLARE_SECRET_KEY"),
		AccountID: os.Getenv("OSS_CLOUDFLARE_ACCOUNT_ID"),
		Region:    os.Getenv("OSS_CLOUDFLARE_REGION"),
		Endpoint:  os.Getenv("OSS_CLOUDFLARE_ENDPOINT"),
		Bucket:    os.Getenv("OSS_CLOUDFLARE_BUCKET"),
		Token:     os.Getenv("OSS_CLOUDFLARE_TOKEN"),
	}
	if config.AccessKey == "" || config.SecretKey == "" || config.Bucket == "" {
		return oss.Config{}, false
	}
	if config.AccountID == "" && config.Endpoint == "" {
		return oss.Config{}, false
	}
	return config, true
}

// skipIfNoEnv skips the current test when R2 credentials are not configured.
func skipIfNoEnv(t *testing.T) oss.Config {
	t.Helper()
	config, ok := testConfig()
	if !ok {
		t.Skip("OSS_CLOUDFLARE_ACCESS_KEY / OSS_CLOUDFLARE_SECRET_KEY / OSS_CLOUDFLARE_BUCKET and account or endpoint are not set, skipping integration test")
	}
	return config
}

// newTestClient creates an R2 client, or skips when credentials are missing.
func newTestClient(t *testing.T) (oss.Storage, oss.Config) {
	t.Helper()
	config := skipIfNoEnv(t)
	client, err := oss.New(oss.ProviderCloudflare, config)
	if err != nil {
		t.Fatalf("oss.New: %v", err)
	}
	return client, config
}
