package baidu

import (
	"errors"
	"net/http"
	"testing"

	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/stretchr/testify/assert"

	"github.com/SmilingXinyi/gb/oss"
)

// TestBucket_Fallback bucket() 在传入空字符串时应回退到 cfg.Bucket。
func TestBucket_Fallback(t *testing.T) {
	a := &adapter{cfg: oss.Config{Bucket: "default-bucket"}}
	assert.Equal(t, "default-bucket", a.bucket(""))
}

// TestBucket_Override bucket() 在传入非空字符串时应使用传入值，忽略 cfg.Bucket。
func TestBucket_Override(t *testing.T) {
	a := &adapter{cfg: oss.Config{Bucket: "default-bucket"}}
	assert.Equal(t, "my-bucket", a.bucket("my-bucket"))
}

// TestIsNotFound_404 BceServiceError with StatusCode 404 should return true.
func TestIsNotFound_404(t *testing.T) {
	err := &bce.BceServiceError{StatusCode: http.StatusNotFound}
	assert.True(t, isNotFound(err))
}

// TestIsNotFound_403 BceServiceError with StatusCode 403 should return false.
func TestIsNotFound_403(t *testing.T) {
	err := &bce.BceServiceError{StatusCode: http.StatusForbidden}
	assert.False(t, isNotFound(err))
}

// TestIsNotFound_NonBce non-BceServiceError should return false.
func TestIsNotFound_NonBce(t *testing.T) {
	assert.False(t, isNotFound(errors.New("some other error")))
}

// TestStatMetadataPrefix Stat 应剥离 BOS 返回的 "x-bce-meta-" 前缀。
// 通过直接操作 UserMeta map 验证 stripMeta 逻辑（white-box）。
func TestStatMetadataPrefix(t *testing.T) {
	input := map[string]string{
		"x-bce-meta-author": "alice",
		"x-bce-meta-env":    "prod",
		"content-type":      "text/plain", // 非 x-bce-meta- 前缀，保持原样
	}
	got := make(map[string]string, len(input))
	for k, v := range input {
		got[stripMetaPrefix(k)] = v
	}
	assert.Equal(t, "alice", got["author"])
	assert.Equal(t, "prod", got["env"])
	assert.Equal(t, "text/plain", got["content-type"])
}
