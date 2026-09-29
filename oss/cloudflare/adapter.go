// Package cloudflare is the Cloudflare R2 adapter for the oss Storage interface.
// It registers itself from init by calling oss.Register(oss.ProviderCloudflare, newAdapter).
// Import the package for side effects to enable the provider:
//
//	import _ "github.com/SmilingXinyi/gb/oss/cloudflare"
//
// R2 speaks the S3 API. This adapter uses aws-sdk-go-v2 with the R2 endpoint and
// the SigV4 region "auto". It is a dedicated provider, separate from oss/s3.
//
// Config field mapping:
//
//	Config.AccessKey → R2 access key ID
//	Config.SecretKey → R2 secret access key
//	Config.Token     → optional session token
//	Config.AccountID → Cloudflare account ID; required when Endpoint is empty
//	Config.Endpoint  → full S3 API URL; overrides the account endpoint
//	Config.Region    → jurisdiction: "" / "auto" (default), "eu", or "fedramp"
//	Config.Bucket    → default bucket (optional)
//
// Endpoint selection when Endpoint is empty:
//
//	auto (default) → https://<AccountID>.r2.cloudflarestorage.com
//	eu             → https://<AccountID>.eu.r2.cloudflarestorage.com
//	fedramp        → https://<AccountID>.fedramp.r2.cloudflarestorage.com
package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/SmilingXinyi/gb/oss"
)

const (
	// signingRegion is the SigV4 region R2 requires. It is not a bucket location.
	signingRegion = "auto"

	// maxPresignSeconds is the longest presigned URL R2 will accept (7 days).
	maxPresignSeconds int64 = 7 * 24 * 60 * 60

	metadataPrefix = "x-amz-meta-"
)

func init() {
	oss.Register(oss.ProviderCloudflare, newAdapter)
}

// adapter implements oss.Storage with a Cloudflare R2 S3 client.
type adapter struct {
	config        oss.Config
	endpoint      string
	client        *s3.Client
	presignClient *s3.PresignClient
}

// newAdapter validates Config and builds an R2 client.
func newAdapter(config oss.Config) (oss.Storage, error) {
	if config.AccessKey == "" {
		return nil, &oss.ErrInvalidConfig{Field: "AccessKey", Message: "required"}
	}
	if config.SecretKey == "" {
		return nil, &oss.ErrInvalidConfig{Field: "SecretKey", Message: "required"}
	}

	endpoint, err := resolveEndpoint(config)
	if err != nil {
		return nil, err
	}

	client := newR2Client(endpoint, config.AccessKey, config.SecretKey, config.Token)
	return &adapter{
		config:        config,
		endpoint:      endpoint,
		client:        client,
		presignClient: s3.NewPresignClient(client),
	}, nil
}

// newR2Client builds an S3 client pointed at a Cloudflare R2 endpoint.
// Checksums are calculated only when required so R2 does not reject the
// default CRC32 headers sent by newer AWS SDK releases.
func newR2Client(endpoint, accessKey, secretKey, token string) *s3.Client {
	awsConfig := aws.Config{
		Region:      signingRegion,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, token),
	}
	return s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
		options.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		options.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
}

// resolveEndpoint returns the R2 S3 API base URL for config.
func resolveEndpoint(config oss.Config) (string, error) {
	if endpoint := strings.TrimSpace(config.Endpoint); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}

	accountID := strings.TrimSpace(config.AccountID)
	if err := validateAccountID(accountID); err != nil {
		return "", err
	}

	host := accountEndpointHost(accountID, config.Region)
	if host == "" {
		return "", &oss.ErrInvalidConfig{
			Field:   "Region",
			Message: "unsupported Cloudflare jurisdiction; use auto, eu, fedramp, or set Endpoint",
		}
	}
	return "https://" + host, nil
}

// accountEndpointHost maps a jurisdiction to the R2 S3 hostname.
// An empty host means the jurisdiction is not supported.
func accountEndpointHost(accountID, jurisdiction string) string {
	switch strings.ToLower(strings.TrimSpace(jurisdiction)) {
	case "", "auto", "default":
		return accountID + ".r2.cloudflarestorage.com"
	case "eu":
		return accountID + ".eu.r2.cloudflarestorage.com"
	case "fedramp":
		return accountID + ".fedramp.r2.cloudflarestorage.com"
	default:
		return ""
	}
}

// validateAccountID checks that accountID can be used as a DNS label.
func validateAccountID(accountID string) error {
	if accountID == "" {
		return &oss.ErrInvalidConfig{Field: "AccountID", Message: "required when Endpoint is empty"}
	}
	if len(accountID) > 63 {
		return &oss.ErrInvalidConfig{Field: "AccountID", Message: "must be at most 63 characters"}
	}
	for _, character := range accountID {
		isDigit := character >= '0' && character <= '9'
		isLower := character >= 'a' && character <= 'z'
		isUpper := character >= 'A' && character <= 'Z'
		if !isDigit && !isLower && !isUpper {
			return &oss.ErrInvalidConfig{Field: "AccountID", Message: "must contain only letters and digits"}
		}
	}
	return nil
}

// normalizeEndpoint accepts a host or absolute URL and returns an absolute URL.
func normalizeEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimRight(endpoint, "/")
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", &oss.ErrInvalidConfig{Field: "Endpoint", Message: "must be an absolute http(s) URL"}
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

// resolveBucket returns the call bucket, or the configured default bucket.
func (receiver *adapter) resolveBucket(bucket string) (string, error) {
	if bucket == "" {
		bucket = receiver.config.Bucket
	}
	if bucket == "" {
		return "", &oss.ErrInvalidConfig{Field: "Bucket", Message: "required"}
	}
	return bucket, nil
}

// requireKey reports an invalid config when an object key is missing.
func requireKey(key string) error {
	if key == "" {
		return &oss.ErrInvalidConfig{Field: "Key", Message: "required"}
	}
	return nil
}

// validatePutOptions rejects options Cloudflare R2 cannot store.
func validatePutOptions(options *oss.PutOptions) error {
	if options == nil {
		return nil
	}
	if options.ACL != "" {
		return &oss.ErrInvalidConfig{Field: "ACL", Message: "Cloudflare R2 does not support object ACLs"}
	}
	return nil
}

// Put uploads an object with PutObject.
// size >= 0 sets Content-Length so non-seekable readers can be uploaded.
// size < 0 leaves Content-Length unset.
func (receiver *adapter) Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, options *oss.PutOptions) error {
	resolvedBucket, err := receiver.resolveBucket(bucket)
	if err != nil {
		return err
	}
	if err = requireKey(key); err != nil {
		return err
	}
	if err = validatePutOptions(options); err != nil {
		return err
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(resolvedBucket),
		Key:    aws.String(key),
		Body:   reader,
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}
	if options != nil {
		if options.ContentType != "" {
			input.ContentType = aws.String(options.ContentType)
		}
		if options.StorageClass != "" {
			input.StorageClass = types.StorageClass(options.StorageClass)
		}
		if len(options.Metadata) > 0 {
			input.Metadata = userMetadata(options.Metadata)
		}
	}

	_, err = receiver.client.PutObject(ctx, input)
	return classifyError(err, "put", resolvedBucket, key)
}

// Get downloads an object body. The caller must close the returned reader.
func (receiver *adapter) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	resolvedBucket, err := receiver.resolveBucket(bucket)
	if err != nil {
		return nil, err
	}
	if err = requireKey(key); err != nil {
		return nil, err
	}

	output, err := receiver.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(resolvedBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, classifyError(err, "get", resolvedBucket, key)
	}
	return output.Body, nil
}

// Delete removes an object.
// R2 usually returns success when the object is already gone. A 404 is mapped
// to ErrObjectNotFound.
func (receiver *adapter) Delete(ctx context.Context, bucket, key string) error {
	resolvedBucket, err := receiver.resolveBucket(bucket)
	if err != nil {
		return err
	}
	if err = requireKey(key); err != nil {
		return err
	}

	_, err = receiver.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(resolvedBucket),
		Key:    aws.String(key),
	})
	return classifyError(err, "delete", resolvedBucket, key)
}

// Stat reads object metadata with HeadObject and does not download the body.
func (receiver *adapter) Stat(ctx context.Context, bucket, key string) (*oss.ObjectMeta, error) {
	resolvedBucket, err := receiver.resolveBucket(bucket)
	if err != nil {
		return nil, err
	}
	if err = requireKey(key); err != nil {
		return nil, err
	}

	output, err := receiver.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(resolvedBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, classifyError(err, "stat", resolvedBucket, key)
	}

	return &oss.ObjectMeta{
		Key:          key,
		Size:         aws.ToInt64(output.ContentLength),
		ContentType:  aws.ToString(output.ContentType),
		ETag:         trimETag(aws.ToString(output.ETag)),
		LastModified: aws.ToTime(output.LastModified),
		StorageClass: string(output.StorageClass),
		Metadata:     normalizeMetadata(output.Metadata),
	}, nil
}

// List lists objects with ListObjectsV2.
// ContinuationToken and NextToken are ListObjectsV2 continuation tokens.
func (receiver *adapter) List(ctx context.Context, bucket, prefix string, options *oss.ListOptions) (*oss.ListResult, error) {
	resolvedBucket, err := receiver.resolveBucket(bucket)
	if err != nil {
		return nil, err
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(resolvedBucket),
		Prefix: aws.String(prefix),
	}
	if options != nil {
		if options.Delimiter != "" {
			input.Delimiter = aws.String(options.Delimiter)
		}
		if options.ContinuationToken != "" {
			input.ContinuationToken = aws.String(options.ContinuationToken)
		}
		if options.MaxKeys < 0 || options.MaxKeys > math.MaxInt32 {
			return nil, &oss.ErrInvalidConfig{Field: "MaxKeys", Message: "must be between 0 and 2147483647"}
		}
		if options.MaxKeys > 0 {
			input.MaxKeys = aws.Int32(int32(options.MaxKeys))
		}
	}

	output, err := receiver.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, classifyError(err, "list", resolvedBucket, prefix)
	}

	result := &oss.ListResult{
		Objects:        []oss.ObjectItem{},
		CommonPrefixes: []string{},
		IsTruncated:    aws.ToBool(output.IsTruncated),
		NextToken:      aws.ToString(output.NextContinuationToken),
	}
	for _, object := range output.Contents {
		result.Objects = append(result.Objects, oss.ObjectItem{
			Key:          aws.ToString(object.Key),
			Size:         aws.ToInt64(object.Size),
			ETag:         trimETag(aws.ToString(object.ETag)),
			LastModified: aws.ToTime(object.LastModified),
			StorageClass: string(object.StorageClass),
		})
	}
	for _, commonPrefix := range output.CommonPrefixes {
		prefixValue := aws.ToString(commonPrefix.Prefix)
		if prefixValue == "" {
			continue
		}
		result.CommonPrefixes = append(result.CommonPrefixes, prefixValue)
	}
	return result, nil
}

// SignURL builds a presigned GET or PUT URL.
// method is case-insensitive. expireSeconds must be between 1 and 7 days.
func (receiver *adapter) SignURL(ctx context.Context, bucket, key, method string, expireSeconds int64) (string, error) {
	resolvedBucket, err := receiver.resolveBucket(bucket)
	if err != nil {
		return "", err
	}
	if err = requireKey(key); err != nil {
		return "", err
	}
	if expireSeconds <= 0 || expireSeconds > maxPresignSeconds {
		return "", &oss.ErrInvalidConfig{Field: "Expire", Message: "must be between 1 and 604800 seconds"}
	}

	expires := time.Duration(expireSeconds) * time.Second
	var presignedURL string
	switch strings.ToUpper(method) {
	case http.MethodGet:
		request, presignErr := receiver.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(resolvedBucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(expires))
		if presignErr != nil {
			return "", classifyError(presignErr, "sign url", resolvedBucket, key)
		}
		presignedURL = request.URL
	case http.MethodPut:
		request, presignErr := receiver.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(resolvedBucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(expires))
		if presignErr != nil {
			return "", classifyError(presignErr, "sign url", resolvedBucket, key)
		}
		presignedURL = request.URL
	default:
		return "", &oss.ErrInvalidConfig{Field: "Method", Message: "must be GET or PUT"}
	}
	return presignedURL, nil
}

// Copy copies an object on the server with CopyObject.
func (receiver *adapter) Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	resolvedSource, err := receiver.resolveBucket(srcBucket)
	if err != nil {
		return err
	}
	resolvedDestination, err := receiver.resolveBucket(dstBucket)
	if err != nil {
		return err
	}
	if err = requireKey(srcKey); err != nil {
		return err
	}
	if err = requireKey(dstKey); err != nil {
		return err
	}

	_, err = receiver.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(resolvedDestination),
		Key:        aws.String(dstKey),
		CopySource: aws.String(formatCopySource(resolvedSource, srcKey)),
	})
	if err != nil {
		return classifyError(err, "copy", resolvedDestination, dstKey)
	}
	return nil
}

// formatCopySource builds the x-amz-copy-source value.
// Path separators stay literal. Every other reserved character is percent-encoded.
func formatCopySource(bucket, key string) string {
	return escapeCopyPath(bucket) + "/" + escapeCopyPath(key)
}

// escapeCopyPath percent-encodes one bucket or key without encoding "/" separators.
func escapeCopyPath(value string) string {
	segments := strings.Split(value, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

// userMetadata strips a caller-supplied x-amz-meta- prefix.
// The AWS SDK adds that prefix again when it sends the request.
func userMetadata(raw map[string]string) map[string]string {
	metadata := make(map[string]string, len(raw))
	for metaKey, metaValue := range raw {
		normalizedKey := metaKey
		if strings.HasPrefix(strings.ToLower(normalizedKey), metadataPrefix) {
			normalizedKey = normalizedKey[len(metadataPrefix):]
		}
		metadata[normalizedKey] = metaValue
	}
	return metadata
}

// normalizeMetadata returns user metadata without the x-amz-meta- prefix.
func normalizeMetadata(raw map[string]string) map[string]string {
	metadata := make(map[string]string, len(raw))
	for metaKey, metaValue := range raw {
		normalizedKey := strings.ToLower(metaKey)
		normalizedKey = strings.TrimPrefix(normalizedKey, metadataPrefix)
		metadata[normalizedKey] = metaValue
	}
	return metadata
}

// trimETag removes the quotes S3-compatible APIs wrap around ETag values.
func trimETag(etag string) string {
	return strings.Trim(etag, `"`)
}

// classifyError maps R2/S3 errors onto the shared oss error types.
func classifyError(err error, operation, bucket, key string) error {
	if err == nil {
		return nil
	}
	if isBucketNotFound(err) {
		return &oss.ErrBucketNotFound{Bucket: bucket}
	}
	if isNotFound(err) {
		return &oss.ErrObjectNotFound{Bucket: bucket, Key: key}
	}
	return fmt.Errorf("oss/cloudflare: %s %s/%s: %w", operation, bucket, key, err)
}

// isBucketNotFound reports whether err is a missing-bucket response.
func isBucketNotFound(err error) bool {
	var apiError smithy.APIError
	return errors.As(err, &apiError) && apiError.ErrorCode() == "NoSuchBucket"
}

// isNotFound reports whether err is a missing-object response.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "NotFound", "NoSuchKey", "NoSuchVersion":
			return true
		}
	}
	return httpStatus(err) == http.StatusNotFound
}

// httpStatus returns the HTTP status carried by err, or 0 when it has none.
func httpStatus(err error) int {
	var responseError *smithyhttp.ResponseError
	if !errors.As(err, &responseError) || responseError.Response == nil || responseError.Response.Response == nil {
		return 0
	}
	return responseError.Response.StatusCode
}
