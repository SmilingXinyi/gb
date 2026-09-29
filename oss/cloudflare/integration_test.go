package cloudflare_test

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

// testKey isolates integration objects under a dedicated prefix.
func testKey(name string) string {
	return fmt.Sprintf("oss-test/%s", name)
}

// TestIntegration_PutAndGet uploads an object and downloads the same bytes.
func TestIntegration_PutAndGet(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	objectKey := testKey("put-get.txt")
	content := []byte("hello, cloudflare r2!")

	err := client.Put(ctx, "", objectKey,
		bytes.NewReader(content), int64(len(content)),
		&oss.PutOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"test-case": "put-and-get"},
		},
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Delete(ctx, "", objectKey) })

	readCloser, err := client.Get(ctx, "", objectKey)
	require.NoError(t, err)
	defer readCloser.Close()

	downloaded, err := io.ReadAll(readCloser)
	require.NoError(t, err)
	assert.Equal(t, content, downloaded)
}

// TestIntegration_Stat checks size, content type, metadata, and ETag.
func TestIntegration_Stat(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	objectKey := testKey("stat.txt")
	content := []byte("stat test content")

	err := client.Put(ctx, "", objectKey,
		bytes.NewReader(content), int64(len(content)),
		&oss.PutOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"test-case": "stat"},
		},
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Delete(ctx, "", objectKey) })

	meta, err := client.Stat(ctx, "", objectKey)
	require.NoError(t, err)

	assert.Equal(t, objectKey, meta.Key)
	assert.Equal(t, int64(len(content)), meta.Size)
	assert.Equal(t, "text/plain", meta.ContentType)
	assert.NotEmpty(t, meta.ETag)
	assert.False(t, meta.LastModified.IsZero())
	assert.Equal(t, "stat", meta.Metadata["test-case"])
}

// TestIntegration_StatNotFound returns ErrObjectNotFound for a missing key.
func TestIntegration_StatNotFound(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	_, err := client.Stat(ctx, "", testKey("__not_exist_object_xyz__.txt"))
	require.Error(t, err)

	var notFound *oss.ErrObjectNotFound
	assert.ErrorAs(t, err, &notFound)
}

// TestIntegration_GetNotFound returns ErrObjectNotFound for a missing key.
func TestIntegration_GetNotFound(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	_, err := client.Get(ctx, "", testKey("__not_exist_object_xyz__.txt"))
	require.Error(t, err)

	var notFound *oss.ErrObjectNotFound
	assert.ErrorAs(t, err, &notFound)
}

// TestIntegration_Delete removes an uploaded object.
func TestIntegration_Delete(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	objectKey := testKey("delete.txt")
	content := []byte("to be deleted")
	err := client.Put(ctx, "", objectKey, bytes.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)

	require.NoError(t, client.Delete(ctx, "", objectKey))

	_, err = client.Stat(ctx, "", objectKey)
	var notFound *oss.ErrObjectNotFound
	assert.ErrorAs(t, err, &notFound)
}

// TestIntegration_List returns every object uploaded under the prefix.
func TestIntegration_List(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	objectKeys := []string{
		testKey("list/a.txt"),
		testKey("list/b.txt"),
		testKey("list/c.txt"),
	}
	for _, objectKey := range objectKeys {
		content := []byte(objectKey)
		err := client.Put(ctx, "", objectKey, bytes.NewReader(content), int64(len(content)), nil)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		for _, objectKey := range objectKeys {
			_ = client.Delete(ctx, "", objectKey)
		}
	})

	result, err := client.List(ctx, "", testKey("list/"), &oss.ListOptions{MaxKeys: 100})
	require.NoError(t, err)
	require.False(t, result.IsTruncated, "expected all objects returned in one page")

	found := make(map[string]bool, len(result.Objects))
	for _, object := range result.Objects {
		found[object.Key] = true
	}
	for _, objectKey := range objectKeys {
		assert.True(t, found[objectKey], "key %q missing from list result", objectKey)
	}
}

// TestIntegration_ListWithDelimiter returns directory-style common prefixes.
func TestIntegration_ListWithDelimiter(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	objectKeys := []string{
		testKey("listdir/dira/1.txt"),
		testKey("listdir/dira/2.txt"),
		testKey("listdir/dirb/1.txt"),
	}
	for _, objectKey := range objectKeys {
		content := []byte(objectKey)
		err := client.Put(ctx, "", objectKey, bytes.NewReader(content), int64(len(content)), nil)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		for _, objectKey := range objectKeys {
			_ = client.Delete(ctx, "", objectKey)
		}
	})

	result, err := client.List(ctx, "", testKey("listdir/"), &oss.ListOptions{
		Delimiter: "/",
		MaxKeys:   100,
	})
	require.NoError(t, err)
	assert.Len(t, result.CommonPrefixes, 2)
	assert.Empty(t, result.Objects)
}

// TestIntegration_SignURL returns a non-empty URL that contains the object key.
func TestIntegration_SignURL(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	objectKey := testKey("sign-url.txt")
	content := []byte("sign url test")
	err := client.Put(ctx, "", objectKey, bytes.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Delete(ctx, "", objectKey) })

	signedURL, err := client.SignURL(ctx, "", objectKey, "GET", 3600)
	require.NoError(t, err)
	assert.NotEmpty(t, signedURL)
	assert.Contains(t, signedURL, "sign-url.txt")
}

// TestIntegration_Copy copies an object on the server and reads the destination.
func TestIntegration_Copy(t *testing.T) {
	client, config := newTestClient(t)
	ctx := context.Background()

	sourceKey := testKey("copy-src.txt")
	destinationKey := testKey("copy-dst.txt")
	content := []byte("copy source content")

	err := client.Put(ctx, "", sourceKey, bytes.NewReader(content), int64(len(content)), nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Delete(ctx, "", sourceKey)
		_ = client.Delete(ctx, "", destinationKey)
	})

	err = client.Copy(ctx, config.Bucket, sourceKey, config.Bucket, destinationKey)
	require.NoError(t, err)

	readCloser, err := client.Get(ctx, "", destinationKey)
	require.NoError(t, err)
	defer readCloser.Close()

	downloaded, err := io.ReadAll(readCloser)
	require.NoError(t, err)
	assert.Equal(t, content, downloaded)
}
