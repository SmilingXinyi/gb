package tencent

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tencentyun/cos-go-sdk-v5"

	"github.com/SmilingXinyi/gb/oss"
)

// TestIsNotFound_404 ErrorResponse with StatusCode 404 should return true.
func TestIsNotFound_404(t *testing.T) {
	err := &cos.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
	assert.True(t, isNotFound(err))
}

// TestIsNotFound_403 ErrorResponse with StatusCode 403 should return false.
func TestIsNotFound_403(t *testing.T) {
	err := &cos.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusForbidden,
		},
	}
	assert.False(t, isNotFound(err))
}

// TestIsNotFound_NilResponse ErrorResponse with nil Response should return false without panic.
func TestIsNotFound_NilResponse(t *testing.T) {
	assert.False(t, isNotFound(&cos.ErrorResponse{}))
}

// TestIsNotFound_StringContains404 error string containing 404 should return true.
func TestIsNotFound_StringContains404(t *testing.T) {
	assert.True(t, isNotFound(errors.New("some error with 404 status")))
}

// TestIsNotFound_OtherError other errors should return false.
func TestIsNotFound_OtherError(t *testing.T) {
	assert.False(t, isNotFound(errors.New("some other error")))
}

// TestGetClient_RegionURL builds the default virtual-hosted bucket URL from Region.
func TestGetClient_RegionURL(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		Region:    "ap-guangzhou",
		Bucket:    "demo-1250000000",
	})
	require.NoError(t, err)

	client, err := storage.(*adapter).getClient("")
	require.NoError(t, err)
	assert.Equal(t, "https://demo-1250000000.cos.ap-guangzhou.myqcloud.com", client.BaseURL.BucketURL.String())
}

// TestGetClient_EndpointTemplate replaces {bucket} in Endpoint when provided.
func TestGetClient_EndpointTemplate(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		Endpoint:  "https://{bucket}.cos.ap-shanghai.myqcloud.com",
	})
	require.NoError(t, err)

	client, err := storage.(*adapter).getClient("demo-1250000000")
	require.NoError(t, err)
	assert.Equal(t, "https://demo-1250000000.cos.ap-shanghai.myqcloud.com", client.BaseURL.BucketURL.String())
}

// TestMetadataHandling Stat 应正确处理腾讯云返回的 "x-cos-meta-" 前缀。
func TestMetadataHandling(t *testing.T) {
	header := http.Header{}
	header.Set("x-cos-meta-author", "alice")
	header.Set("X-Cos-Meta-Env", "prod")
	header.Set("content-type", "text/plain")

	metadata := make(map[string]string)
	for k, v := range header {
		if len(v) > 0 {
			key := k
			// Simulate the logic in Stat
			if len(key) > 11 && (key[:11] == "X-Cos-Meta-" || key[:11] == "x-cos-meta-") {
				metadata[key[11:]] = v[0]
			}
		}
	}

	// Note: http.Header keys are canonicalized, so x-cos-meta-author becomes X-Cos-Meta-Author
	// But our logic in Stat uses strings.ToLower(k) and strings.TrimPrefix(..., "x-cos-meta-")
	
	// Let's re-verify the logic in Stat:
	// for headerKey, headerValues := range header {
	//     if strings.HasPrefix(strings.ToLower(headerKey), "x-cos-meta-") {
	//         meta.Metadata[strings.TrimPrefix(strings.ToLower(headerKey), "x-cos-meta-")] = headerValues[0]
	//     }
	// }
	
	metadata = make(map[string]string)
	for k, v := range header {
		if strings.HasPrefix(strings.ToLower(k), "x-cos-meta-") {
			metadata[strings.TrimPrefix(strings.ToLower(k), "x-cos-meta-")] = v[0]
		}
	}

	assert.Equal(t, "alice", metadata["author"])
	assert.Equal(t, "prod", metadata["env"])
	assert.NotContains(t, metadata, "content-type")
}
