package cloudflare

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SmilingXinyi/gb/oss"
)

// TestResolveEndpoint_AccountID builds the default R2 host from AccountID.
func TestResolveEndpoint_AccountID(t *testing.T) {
	endpoint, err := resolveEndpoint(oss.Config{AccountID: "abc123def"})
	require.NoError(t, err)
	assert.Equal(t, "https://abc123def.r2.cloudflarestorage.com", endpoint)
}

// TestResolveEndpoint_Jurisdictions selects the EU and FedRAMP hosts.
func TestResolveEndpoint_Jurisdictions(t *testing.T) {
	euEndpoint, err := resolveEndpoint(oss.Config{AccountID: "abc123def", Region: "EU"})
	require.NoError(t, err)
	assert.Equal(t, "https://abc123def.eu.r2.cloudflarestorage.com", euEndpoint)

	fedrampEndpoint, err := resolveEndpoint(oss.Config{AccountID: "abc123def", Region: "fedramp"})
	require.NoError(t, err)
	assert.Equal(t, "https://abc123def.fedramp.r2.cloudflarestorage.com", fedrampEndpoint)
}

// TestResolveEndpoint_AutoRegion uses the default host for auto and default.
func TestResolveEndpoint_AutoRegion(t *testing.T) {
	for _, region := range []string{"", "auto", "default"} {
		endpoint, err := resolveEndpoint(oss.Config{AccountID: "abc123def", Region: region})
		require.NoError(t, err, "region=%q", region)
		assert.Equal(t, "https://abc123def.r2.cloudflarestorage.com", endpoint)
	}
}

// TestResolveEndpoint_RejectsUnknownJurisdiction returns ErrInvalidConfig.
func TestResolveEndpoint_RejectsUnknownJurisdiction(t *testing.T) {
	_, err := resolveEndpoint(oss.Config{AccountID: "abc123def", Region: "us-east-1"})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "Region", configErr.Field)
}

// TestResolveEndpoint_ExplicitEndpoint overrides AccountID and Region.
func TestResolveEndpoint_ExplicitEndpoint(t *testing.T) {
	endpoint, err := resolveEndpoint(oss.Config{
		AccountID: "abc123def",
		Region:    "eu",
		Endpoint:  "account.r2.cloudflarestorage.com/",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://account.r2.cloudflarestorage.com", endpoint)
}

// TestResolveEndpoint_RejectsEmptyAccount returns ErrInvalidConfig.
func TestResolveEndpoint_RejectsEmptyAccount(t *testing.T) {
	_, err := resolveEndpoint(oss.Config{})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "AccountID", configErr.Field)
}

// TestResolveEndpoint_RejectsInvalidAccount rejects characters that break the host.
func TestResolveEndpoint_RejectsInvalidAccount(t *testing.T) {
	_, err := resolveEndpoint(oss.Config{AccountID: "not a host"})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "AccountID", configErr.Field)
}

// TestNewAdapter_StoresEndpoint keeps the resolved endpoint on the adapter.
func TestNewAdapter_StoresEndpoint(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		AccountID: "abc123def",
		Bucket:    "demo",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://abc123def.r2.cloudflarestorage.com", storage.(*adapter).endpoint)
}

// TestResolveBucket_UsesDefault falls back to the configured bucket.
func TestResolveBucket_UsesDefault(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		Endpoint:  "https://example.r2.cloudflarestorage.com",
		Bucket:    "demo",
	})
	require.NoError(t, err)

	bucket, err := storage.(*adapter).resolveBucket("")
	require.NoError(t, err)
	assert.Equal(t, "demo", bucket)
}

// TestResolveBucket_RequiresBucket when neither argument nor config has one.
func TestResolveBucket_RequiresBucket(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		Endpoint:  "https://example.r2.cloudflarestorage.com",
	})
	require.NoError(t, err)

	_, err = storage.(*adapter).resolveBucket("")
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "Bucket", configErr.Field)
}

// TestValidatePutOptions_RejectsACL because R2 does not support object ACLs.
func TestValidatePutOptions_RejectsACL(t *testing.T) {
	err := validatePutOptions(&oss.PutOptions{ACL: "public-read"})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "ACL", configErr.Field)
	assert.NoError(t, validatePutOptions(nil))
}

// TestFormatCopySource encodes reserved characters and keeps path separators.
func TestFormatCopySource(t *testing.T) {
	assert.Equal(t, "demo/docs/a.txt", formatCopySource("demo", "docs/a.txt"))
	assert.Equal(t, "demo/my%20file.txt", formatCopySource("demo", "my file.txt"))
}

// TestUserMetadata_StripsPrefix removes an x-amz-meta- prefix before upload.
func TestUserMetadata_StripsPrefix(t *testing.T) {
	metadata := userMetadata(map[string]string{
		"author":          "alice",
		"x-amz-meta-env":  "prod",
		"X-Amz-Meta-Team": "oss",
	})
	assert.Equal(t, "alice", metadata["author"])
	assert.Equal(t, "prod", metadata["env"])
	assert.Equal(t, "oss", metadata["Team"])
}

// TestNormalizeMetadata lowercases keys and drops the provider prefix.
func TestNormalizeMetadata(t *testing.T) {
	metadata := normalizeMetadata(map[string]string{
		"Author":         "alice",
		"x-amz-meta-env": "prod",
	})
	assert.Equal(t, "alice", metadata["author"])
	assert.Equal(t, "prod", metadata["env"])
}

// TestTrimETag removes surrounding quotes.
func TestTrimETag(t *testing.T) {
	assert.Equal(t, "abc", trimETag(`"abc"`))
	assert.Equal(t, "abc", trimETag("abc"))
}

// TestIsNotFound_APIError recognizes S3 missing-object codes.
func TestIsNotFound_APIError(t *testing.T) {
	assert.True(t, isNotFound(&smithy.GenericAPIError{Code: "NoSuchKey", Message: "missing"}))
	assert.True(t, isNotFound(&smithy.GenericAPIError{Code: "NotFound", Message: "missing"}))
	assert.False(t, isNotFound(&smithy.GenericAPIError{Code: "AccessDenied", Message: "no"}))
	assert.False(t, isNotFound(nil))
}

// TestIsNotFound_HTTPStatus treats a bare 404 as a missing object.
func TestIsNotFound_HTTPStatus(t *testing.T) {
	err := &smithyhttp.ResponseError{
		Response: &smithyhttp.Response{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		},
		Err: errors.New("missing"),
	}
	assert.True(t, isNotFound(err))
}

// TestClassifyError_BucketNotFound prefers ErrBucketNotFound over object 404.
func TestClassifyError_BucketNotFound(t *testing.T) {
	err := classifyError(&smithy.GenericAPIError{Code: "NoSuchBucket", Message: "missing"}, "get", "demo", "a.txt")
	var bucketErr *oss.ErrBucketNotFound
	assert.ErrorAs(t, err, &bucketErr)
	assert.Equal(t, "demo", bucketErr.Bucket)
}

// TestClassifyError_ObjectNotFound maps NoSuchKey onto ErrObjectNotFound.
func TestClassifyError_ObjectNotFound(t *testing.T) {
	err := classifyError(&smithy.GenericAPIError{Code: "NoSuchKey", Message: "missing"}, "get", "demo", "a.txt")
	var notFound *oss.ErrObjectNotFound
	assert.ErrorAs(t, err, &notFound)
	assert.Equal(t, "demo", notFound.Bucket)
	assert.Equal(t, "a.txt", notFound.Key)
}

// TestSignURL_PresignsLocally builds a path-style URL without calling the network.
func TestSignURL_PresignsLocally(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		AccountID: "abc123def",
		Bucket:    "demo",
	})
	require.NoError(t, err)

	signedURL, err := storage.SignURL(context.Background(), "", "docs/a.txt", "get", 60)
	require.NoError(t, err)
	assert.Contains(t, signedURL, "https://abc123def.r2.cloudflarestorage.com/demo/docs/a.txt")
	assert.Contains(t, signedURL, "X-Amz-Algorithm")
	assert.Contains(t, signedURL, "X-Amz-Signature")

	putURL, err := storage.SignURL(context.Background(), "demo", "docs/a.txt", "PUT", 60)
	require.NoError(t, err)
	assert.Contains(t, putURL, "X-Amz-Signature")
}

// TestSignURL_RejectsBadInput validates method and expiration locally.
func TestSignURL_RejectsBadInput(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		AccountID: "abc123def",
		Bucket:    "demo",
	})
	require.NoError(t, err)

	_, err = storage.SignURL(context.Background(), "demo", "a.txt", "DELETE", 60)
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "Method", configErr.Field)

	_, err = storage.SignURL(context.Background(), "demo", "a.txt", "GET", 0)
	require.Error(t, err)
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "Expire", configErr.Field)
}

// TestPut_RejectsACL before any network call.
func TestPut_RejectsACL(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		AccountID: "abc123def",
		Bucket:    "demo",
	})
	require.NoError(t, err)

	err = storage.Put(context.Background(), "demo", "a.txt", nil, 0, &oss.PutOptions{ACL: "private"})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "ACL", configErr.Field)
}

// TestList_RejectsNegativeMaxKeys before any network call.
func TestList_RejectsNegativeMaxKeys(t *testing.T) {
	storage, err := newAdapter(oss.Config{
		AccessKey: "ak",
		SecretKey: "sk",
		AccountID: "abc123def",
		Bucket:    "demo",
	})
	require.NoError(t, err)

	_, err = storage.List(context.Background(), "demo", "", &oss.ListOptions{MaxKeys: -1})
	require.Error(t, err)
	var configErr *oss.ErrInvalidConfig
	assert.ErrorAs(t, err, &configErr)
	assert.Equal(t, "MaxKeys", configErr.Field)
}
