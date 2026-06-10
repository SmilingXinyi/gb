// Package tencent 是腾讯云 COS（Cloud Object Storage）的 Storage 适配器。
// 通过在 init() 中调用 oss.Register(oss.ProviderTencent, newAdapter) 完成自动注册。
//
// 依赖 SDK：github.com/tencentyun/cos-go-sdk-v5
//
// 配置字段映射：
//
//	Config.AccessKey  → SecretID
//	Config.SecretKey  → SecretKey
//	Config.Token      → SessionToken（临时凭证时使用）
//	Config.Region     → COS 地域（如 "ap-guangzhou"）
//	Config.Endpoint   → 若非空，直接作为 BucketURL（格式 "https://{bucket}.cos.{region}.myqcloud.com"）
//	Config.Bucket     → 默认 bucket（格式 "{name}-{appid}"，可选）
package tencent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SmilingXinyi/gb/oss"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func init() {
	oss.Register(oss.ProviderTencent, newAdapter)
}

// adapter 实现 oss.Storage 接口，封装腾讯云 COS SDK 客户端。
type adapter struct {
	config oss.Config
}

// newAdapter 根据 Config 创建并初始化腾讯云 COS adapter。
func newAdapter(config oss.Config) (oss.Storage, error) {
	if config.AccessKey == "" {
		return nil, &oss.ErrInvalidConfig{Field: "AccessKey", Message: "required"}
	}
	if config.SecretKey == "" {
		return nil, &oss.ErrInvalidConfig{Field: "SecretKey", Message: "required"}
	}
	if config.Region == "" && config.Endpoint == "" {
		return nil, &oss.ErrInvalidConfig{Field: "Region", Message: "required when Endpoint is empty"}
	}

	return &adapter{config: config}, nil
}

// getClient 根据 bucket 返回一个 cos.Client 实例。
func (receiver *adapter) getClient(bucket string) (*cos.Client, error) {
	bucketName := bucket
	if bucketName == "" {
		bucketName = receiver.config.Bucket
	}

	var bucketURLStr string
	if receiver.config.Endpoint != "" {
		// 如果提供了 Endpoint，假设它是完整的 BucketURL 模板或直接使用
		// 注意：腾讯云 SDK 通常需要针对每个 bucket 实例化 client
		bucketURLStr = receiver.config.Endpoint
		if bucket != "" && strings.Contains(bucketURLStr, "{bucket}") {
			bucketURLStr = strings.ReplaceAll(bucketURLStr, "{bucket}", bucket)
		}
	} else {
		bucketURLStr = fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucketName, receiver.config.Region)
	}

	parsedURL, err := url.Parse(bucketURLStr)
	if err != nil {
		return nil, fmt.Errorf("oss/tencent: parse bucket url %q: %w", bucketURLStr, err)
	}

	baseURL := &cos.BaseURL{BucketURL: parsedURL}
	httpClient := &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     receiver.config.AccessKey,
			SecretKey:    receiver.config.SecretKey,
			SessionToken: receiver.config.Token,
		},
	}

	return cos.NewClient(baseURL, httpClient), nil
}

// Put 调用 COS SDK 的 Object.Put 上传对象。
func (receiver *adapter) Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, options *oss.PutOptions) error {
	client, err := receiver.getClient(bucket)
	if err != nil {
		return err
	}

	putOptions := &cos.ObjectPutOptions{}
	if options != nil {
		putOptions.ObjectPutHeaderOptions = &cos.ObjectPutHeaderOptions{
			ContentType:  options.ContentType,
			XCosStorageClass: options.StorageClass,
		}
		if len(options.Metadata) > 0 {
			putOptions.XCosMetaXXX = &http.Header{}
			for metaKey, metaValue := range options.Metadata {
				// 腾讯云自定义元数据前缀为 x-cos-meta-
				fullKey := metaKey
				if !strings.HasPrefix(strings.ToLower(metaKey), "x-cos-meta-") {
					fullKey = "x-cos-meta-" + metaKey
				}
				putOptions.XCosMetaXXX.Add(fullKey, metaValue)
			}
		}
		if options.ACL != "" {
			putOptions.ACLHeaderOptions = &cos.ACLHeaderOptions{
				XCosACL: options.ACL,
			}
		}
	}

	_, err = client.Object.Put(ctx, key, reader, putOptions)
	if err != nil {
		return fmt.Errorf("oss/tencent: put %s: %w", key, err)
	}
	return nil
}

// Get 调用 COS SDK 的 Object.Get 下载对象。
func (receiver *adapter) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	client, err := receiver.getClient(bucket)
	if err != nil {
		return nil, err
	}

	response, err := client.Object.Get(ctx, key, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, &oss.ErrObjectNotFound{Bucket: bucket, Key: key}
		}
		return nil, fmt.Errorf("oss/tencent: get %s: %w", key, err)
	}
	return response.Body, nil
}

// Delete 调用 COS SDK 的 Object.Delete 删除对象。
func (receiver *adapter) Delete(ctx context.Context, bucket, key string) error {
	client, err := receiver.getClient(bucket)
	if err != nil {
		return err
	}

	_, err = client.Object.Delete(ctx, key, nil)
	if err != nil {
		if isNotFound(err) {
			return &oss.ErrObjectNotFound{Bucket: bucket, Key: key}
		}
		return fmt.Errorf("oss/tencent: delete %s: %w", key, err)
	}
	return nil
}

// Stat 调用 COS SDK 的 Object.Head 获取对象元信息。
func (receiver *adapter) Stat(ctx context.Context, bucket, key string) (*oss.ObjectMeta, error) {
	client, err := receiver.getClient(bucket)
	if err != nil {
		return nil, err
	}

	response, err := client.Object.Head(ctx, key, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, &oss.ErrObjectNotFound{Bucket: bucket, Key: key}
		}
		return nil, fmt.Errorf("oss/tencent: stat %s: %w", key, err)
	}

	header := response.Header
	lastModified, _ := http.ParseTime(header.Get("Last-Modified"))
	
	meta := &oss.ObjectMeta{
		Key:          key,
		Size:         response.ContentLength,
		ContentType:  header.Get("Content-Type"),
		ETag:         strings.Trim(header.Get("ETag"), `"`),
		LastModified: lastModified,
		StorageClass: header.Get("X-Cos-Storage-Class"),
		Metadata:     make(map[string]string),
	}

	for headerKey, headerValues := range header {
		if strings.HasPrefix(strings.ToLower(headerKey), "x-cos-meta-") {
			meta.Metadata[strings.TrimPrefix(strings.ToLower(headerKey), "x-cos-meta-")] = headerValues[0]
		}
	}

	return meta, nil
}

// List 调用 COS SDK 的 Bucket.Get 列举对象.
func (receiver *adapter) List(ctx context.Context, bucket, prefix string, options *oss.ListOptions) (*oss.ListResult, error) {
	client, err := receiver.getClient(bucket)
	if err != nil {
		return nil, err
	}

	listOptions := &cos.BucketGetOptions{
		Prefix: prefix,
	}
	if options != nil {
		listOptions.Delimiter = options.Delimiter
		listOptions.Marker = options.ContinuationToken
		if options.MaxKeys > 0 {
			listOptions.MaxKeys = options.MaxKeys
		}
	}

	listResult, _, err := client.Bucket.Get(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("oss/tencent: list %s: %w", prefix, err)
	}

	result := &oss.ListResult{
		IsTruncated: listResult.IsTruncated,
		NextToken:   listResult.NextMarker,
	}

	for _, object := range listResult.Contents {
		lastModified, _ := time.Parse(time.RFC3339, object.LastModified)
		result.Objects = append(result.Objects, oss.ObjectItem{
			Key:          object.Key,
			Size:         int64(object.Size),
			ETag:         strings.Trim(object.ETag, `"`),
			LastModified: lastModified,
			StorageClass: object.StorageClass,
		})
	}

	result.CommonPrefixes = listResult.CommonPrefixes

	return result, nil
}

// SignURL 调用 COS SDK 的 Object.GetPresignedURL 生成预签名 URL。
func (receiver *adapter) SignURL(ctx context.Context, bucket, key, method string, expireSeconds int64) (string, error) {
	client, err := receiver.getClient(bucket)
	if err != nil {
		return "", err
	}

	expireDuration := time.Duration(expireSeconds) * time.Second
	presignedURL, err := client.Object.GetPresignedURL(ctx, method, key, receiver.config.AccessKey, receiver.config.SecretKey, expireDuration, nil)
	if err != nil {
		return "", fmt.Errorf("oss/tencent: sign url %s: %w", key, err)
	}

	return presignedURL.String(), nil
}

// Copy 调用 COS SDK 的 Object.Copy 实现服务端复制。
func (receiver *adapter) Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	client, err := receiver.getClient(dstBucket)
	if err != nil {
		return err
	}

	// 腾讯云 Copy 需要源对象的 URL
	// 格式: <bucketname-appid>.cos.<region>.myqcloud.com/<key>
	srcBucketName := srcBucket
	if srcBucketName == "" {
		srcBucketName = receiver.config.Bucket
	}
	
	srcURL := fmt.Sprintf("%s.cos.%s.myqcloud.com/%s", srcBucketName, receiver.config.Region, srcKey)
	
	_, _, err = client.Object.Copy(ctx, dstKey, srcURL, nil)
	if err != nil {
		return fmt.Errorf("oss/tencent: copy %s/%s to %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}
	return nil
}

// isNotFound 判断是否为 404 错误。
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// 腾讯云 SDK 错误处理
	if cosErr, ok := err.(*cos.ErrorResponse); ok {
		return cosErr.Response.StatusCode == http.StatusNotFound
	}
	return strings.Contains(err.Error(), "404")
}
