package baidu_test

import (
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/SmilingXinyi/gb/oss"
	_ "github.com/SmilingXinyi/gb/oss/baidu"
)

// TestMain 在所有测试运行前统一加载 .env，避免每个测试重复加载。
func TestMain(m *testing.M) {
	_ = godotenv.Load(".env")
	os.Exit(m.Run())
}

// testConfig 读取环境变量，构造 oss.Config。
// 若必填字段缺失则返回 ("", false)，调用方应调用 t.Skip。
func testConfig() (oss.Config, bool) {
	cfg := oss.Config{
		AccessKey: os.Getenv("BAIDU_OSS_AK"),
		SecretKey: os.Getenv("BAIDU_OSS_SK"),
		Region:    os.Getenv("BAIDU_OSS_REGION"),
		Bucket:    os.Getenv("BAIDU_OSS_BUCKET"),
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
		t.Skip("BAIDU_OSS_AK / BAIDU_OSS_SK / BAIDU_OSS_BUCKET not set, skipping integration test")
	}
	return cfg
}

// newTestClient 创建测试用 BOS 客户端，凭证缺失时自动 skip。
func newTestClient(t *testing.T) (oss.Storage, oss.Config) {
	t.Helper()
	cfg := skipIfNoEnv(t)
	client, err := oss.New(oss.ProviderBaidu, cfg)
	if err != nil {
		t.Fatalf("oss.New: %v", err)
	}
	return client, cfg
}
