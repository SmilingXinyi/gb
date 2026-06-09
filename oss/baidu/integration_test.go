package baidu_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SmilingXinyi/gb/oss"
)

// testKey 生成带前缀的对象 key，与业务数据隔离，方便批量清理。
func testKey(name string) string {
	return fmt.Sprintf("oss-test/%s", name)
}

// TestIntegration_PutAndGet 上传后下载，验证内容一致。
func TestIntegration_PutAndGet(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	key := testKey("put-get.txt")
	content := []byte("hello, baidu oss!")

	err := client.Put(ctx, "", key,
		bytes.NewReader(content), int64(len(content)),
		&oss.PutOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"test-case": "put-and-get"},
		},
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Delete(ctx, "", key) })

	rc, err := client.Get(ctx, "", key)
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

// TestIntegration_Stat 上传后获取元信息，验证 Size、ContentType、ETag、LastModified 字段。
func TestIntegration_Stat(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	key := testKey("stat.txt")
	content := []byte("stat test content")

	err := client.Put(ctx, "", key,
		bytes.NewReader(content), int64(len(content)),
		&oss.PutOptions{ContentType: "text/plain"},
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Delete(ctx, "", key) })

	meta, err := client.Stat(ctx, "", key)
	require.NoError(t, err)

	assert.Equal(t, key, meta.Key)
	assert.Equal(t, int64(len(content)), meta.Size)
	assert.Equal(t, "text/plain", meta.ContentType)
	assert.NotEmpty(t, meta.ETag)
	assert.False(t, meta.LastModified.IsZero())
}

// TestIntegration_StatNotFound 对不存在的对象调用 Stat，应返回 ErrObjectNotFound。
func TestIntegration_StatNotFound(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	_, err := client.Stat(ctx, "", testKey("__not_exist_object_xyz__.txt"))
	require.Error(t, err)

	var notFound *oss.ErrObjectNotFound
	assert.ErrorAs(t, err, &notFound)
}

// TestIntegration_GetNotFound 对不存在的对象调用 Get，应返回 ErrObjectNotFound。
func TestIntegration_GetNotFound(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	_, err := client.Get(ctx, "", testKey("__not_exist_object_xyz__.txt"))
	require.Error(t, err)

	var notFound *oss.ErrObjectNotFound
	assert.ErrorAs(t, err, &notFound)
}

// TestIntegration_Delete 上传后删除，再 Stat 应返回 ErrObjectNotFound。
func TestIntegration_Delete(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	key := testKey("delete.txt")
	err := client.Put(ctx, "", key,
		bytes.NewReader([]byte("to be deleted")), 13, nil,
	)
	require.NoError(t, err)

	require.NoError(t, client.Delete(ctx, "", key))

	_, err = client.Stat(ctx, "", key)
	var notFound *oss.ErrObjectNotFound
	assert.ErrorAs(t, err, &notFound)
}

// TestIntegration_List 上传多个对象后列举，验证结果中包含所有已上传的 key。
func TestIntegration_List(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	keys := []string{
		testKey("list/a.txt"),
		testKey("list/b.txt"),
		testKey("list/c.txt"),
	}
	for _, k := range keys {
		err := client.Put(ctx, "", k, bytes.NewReader([]byte(k)), int64(len(k)), nil)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		for _, k := range keys {
			_ = client.Delete(ctx, "", k)
		}
	})

	result, err := client.List(ctx, "", testKey("list/"), &oss.ListOptions{MaxKeys: 100})
	require.NoError(t, err)
	require.False(t, result.IsTruncated, "expected all objects returned in one page")

	got := make(map[string]bool, len(result.Objects))
	for _, obj := range result.Objects {
		got[obj.Key] = true
	}
	for _, k := range keys {
		assert.True(t, got[k], "key %q missing from list result", k)
	}
}

// TestIntegration_ListWithDelimiter 使用 Delimiter 列举，验证 CommonPrefixes 模拟目录。
func TestIntegration_ListWithDelimiter(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	// 上传两个"子目录"下的文件
	keys := []string{
		testKey("listdir/dira/1.txt"),
		testKey("listdir/dira/2.txt"),
		testKey("listdir/dirb/1.txt"),
	}
	for _, k := range keys {
		err := client.Put(ctx, "", k, bytes.NewReader([]byte(k)), int64(len(k)), nil)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		for _, k := range keys {
			_ = client.Delete(ctx, "", k)
		}
	})

	result, err := client.List(ctx, "", testKey("listdir/"), &oss.ListOptions{
		Delimiter: "/",
		MaxKeys:   100,
	})
	require.NoError(t, err)

	// 应有 2 个公共前缀（dira/ dirb/），不直接返回文件
	assert.Len(t, result.CommonPrefixes, 2)
	assert.Empty(t, result.Objects)
}

// TestIntegration_SignURL 生成预签名下载 URL，验证非空且包含对象 key。
func TestIntegration_SignURL(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	key := testKey("sign-url.txt")
	content := []byte("sign url test")
	err := client.Put(ctx, "", key, bytes.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Delete(ctx, "", key) })

	url, err := client.SignURL(ctx, "", key, "GET", 3600)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, key)
}

// TestIntegration_Copy 服务端复制对象，验证目标对象内容与源一致。
func TestIntegration_Copy(t *testing.T) {
	client, cfg := newTestClient(t)
	ctx := context.Background()

	srcKey := testKey("copy-src.txt")
	dstKey := testKey("copy-dst.txt")
	content := []byte("copy source content")

	err := client.Put(ctx, "", srcKey, bytes.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Delete(ctx, "", srcKey)
		_ = client.Delete(ctx, "", dstKey)
	})

	err = client.Copy(ctx, cfg.Bucket, srcKey, cfg.Bucket, dstKey)
	require.NoError(t, err)

	rc, err := client.Get(ctx, "", dstKey)
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}
