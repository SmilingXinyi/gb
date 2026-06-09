// Package baidu 是百度云 BOS（Baidu Object Storage）的 Storage 适配器。
// 通过在 init() 中调用 oss.Register(oss.ProviderBaidu, newAdapter) 完成自动注册，
// 主程序只需 import _ "…/oss/baidu" 即可激活百度云 provider。
//
// 依赖：github.com/baidubce/bce-sdk-go v0.9.x
//
// 配置字段映射：
//
//	Config.AccessKey → BOS AK
//	Config.SecretKey → BOS SK
//	Config.Token     → STS SessionToken（临时凭证时使用）
//	Config.Region    → BOS 地域前缀，用于推导 Endpoint（如 "bj" → "bj.bcebos.com"）
//	Config.Endpoint  → 若非空，直接使用（覆盖 Region 推导结果）
//	Config.Bucket    → 默认 bucket（可选）
package baidu

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/baidubce/bce-sdk-go/auth"
	"github.com/baidubce/bce-sdk-go/bce"
	bosapi "github.com/baidubce/bce-sdk-go/services/bos/api"

	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/SmilingXinyi/gb/oss"
)

func init() {
	oss.Register(oss.ProviderBaidu, newAdapter)
}

// adapter 实现 oss.Storage 接口，内部持有 BOS SDK 客户端。
type adapter struct {
	client *bos.Client
	cfg    oss.Config
}

// newAdapter 根据 Config 创建并初始化 BOS adapter。
func newAdapter(cfg oss.Config) (oss.Storage, error) {
	if cfg.AccessKey == "" {
		return nil, &oss.ErrInvalidConfig{Field: "AccessKey", Message: "required"}
	}
	if cfg.SecretKey == "" {
		return nil, &oss.ErrInvalidConfig{Field: "SecretKey", Message: "required"}
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		if cfg.Region == "" {
			cfg.Region = "bj" // 默认北京
		}
		endpoint = fmt.Sprintf("%s.bcebos.com", cfg.Region)
	}

	client, err := bos.NewClient(cfg.AccessKey, cfg.SecretKey, endpoint)
	if err != nil {
		return nil, fmt.Errorf("oss/baidu: init client: %w", err)
	}

	// 使用 STS 临时凭证
	if cfg.Token != "" {
		stsCred, credErr := auth.NewSessionBceCredentials(cfg.AccessKey, cfg.SecretKey, cfg.Token)
		if credErr != nil {
			return nil, fmt.Errorf("oss/baidu: init sts credentials: %w", credErr)
		}
		client.Config.Credentials = stsCred
	}

	return &adapter{client: client, cfg: cfg}, nil
}

// bucket 返回调用时指定的 bucket；若为空则回退到配置的默认 bucket。
func (a *adapter) bucket(bucket string) string {
	if bucket != "" {
		return bucket
	}
	return a.cfg.Bucket
}

// Put 调用 BOS SDK PutObjectFromStream 上传对象。
func (a *adapter) Put(_ context.Context, bucket, key string, reader io.Reader, size int64, opts *oss.PutOptions) error {
	bucket = a.bucket(bucket)

	body, err := bce.NewBodyFromSizedReader(reader, size)
	if err != nil {
		return fmt.Errorf("oss/baidu: build body: %w", err)
	}

	var args *bosapi.PutObjectArgs
	if opts != nil {
		args = &bosapi.PutObjectArgs{
			ContentType:  opts.ContentType,
			StorageClass: opts.StorageClass,
			UserMeta:     opts.Metadata,
		}
		if opts.ACL != "" {
			// BOS 通过 CannedAcl 设置 ACL，此处通过 ObjectAcl 参数传递
			args.CannedAcl = opts.ACL
		}
	}

	_, err = a.client.PutObject(bucket, key, body, args)
	if err != nil {
		return fmt.Errorf("oss/baidu: put %s/%s: %w", bucket, key, err)
	}
	return nil
}

// Get 调用 BOS SDK BasicGetObject 下载对象，返回响应体 ReadCloser。
func (a *adapter) Get(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	bucket = a.bucket(bucket)

	res, err := a.client.BasicGetObject(bucket, key)
	if err != nil {
		if isNotFound(err) {
			return nil, &oss.ErrObjectNotFound{Bucket: bucket, Key: key}
		}
		return nil, fmt.Errorf("oss/baidu: get %s/%s: %w", bucket, key, err)
	}
	return res.Body, nil
}

// Delete 调用 BOS SDK DeleteObject 删除对象。
func (a *adapter) Delete(_ context.Context, bucket, key string) error {
	bucket = a.bucket(bucket)

	if err := a.client.DeleteObject(bucket, key); err != nil {
		if isNotFound(err) {
			return &oss.ErrObjectNotFound{Bucket: bucket, Key: key}
		}
		return fmt.Errorf("oss/baidu: delete %s/%s: %w", bucket, key, err)
	}
	return nil
}

// Stat 调用 BOS SDK GetObjectMeta 获取对象元信息，不传输对象内容。
func (a *adapter) Stat(_ context.Context, bucket, key string) (*oss.ObjectMeta, error) {
	bucket = a.bucket(bucket)

	res, err := a.client.GetObjectMeta(bucket, key)
	if err != nil {
		if isNotFound(err) {
			return nil, &oss.ErrObjectNotFound{Bucket: bucket, Key: key}
		}
		return nil, fmt.Errorf("oss/baidu: stat %s/%s: %w", bucket, key, err)
	}

	lastModified, err := http.ParseTime(res.LastModified)
	if err != nil {
		return nil, fmt.Errorf("oss/baidu: stat %s/%s: parse LastModified %q: %w", bucket, key, res.LastModified, err)
	}
	meta := &oss.ObjectMeta{
		Key:          key,
		Size:         res.ContentLength,
		ContentType:  res.ContentType,
		ETag:         strings.Trim(res.ETag, `"`),
		LastModified: lastModified,
		StorageClass: res.StorageClass,
	}
	// 去除 BOS 自定义元数据前缀 "x-bce-meta-"
	if len(res.UserMeta) > 0 {
		meta.Metadata = make(map[string]string, len(res.UserMeta))
		for k, v := range res.UserMeta {
			meta.Metadata[stripMetaPrefix(k)] = v
		}
	}
	return meta, nil
}

// List 调用 BOS SDK ListObjects 列举对象，支持分页与 Delimiter。
func (a *adapter) List(_ context.Context, bucket, prefix string, opts *oss.ListOptions) (*oss.ListResult, error) {
	bucket = a.bucket(bucket)

	args := &bosapi.ListObjectsArgs{
		Prefix: prefix,
	}
	if opts != nil {
		args.Delimiter = opts.Delimiter
		args.Marker = opts.ContinuationToken // BOS v1 使用 Marker 分页
		if opts.MaxKeys > 0 {
			args.MaxKeys = opts.MaxKeys
		}
	}

	res, err := a.client.ListObjects(bucket, args)
	if err != nil {
		return nil, fmt.Errorf("oss/baidu: list %s/%s: %w", bucket, prefix, err)
	}

	result := &oss.ListResult{
		IsTruncated: res.IsTruncated,
		NextToken:   res.NextMarker,
	}
	for _, obj := range res.Contents {
		lastModified, err := time.Parse(time.RFC3339, obj.LastModified)
		if err != nil {
			return nil, fmt.Errorf("oss/baidu: list %s/%s: parse LastModified %q for key %q: %w", bucket, prefix, obj.LastModified, obj.Key, err)
		}
		result.Objects = append(result.Objects, oss.ObjectItem{
			Key:          obj.Key,
			Size:         int64(obj.Size),
			ETag:         strings.Trim(obj.ETag, `"`),
			LastModified: lastModified,
			StorageClass: obj.StorageClass,
		})
	}
	for _, p := range res.CommonPrefixes {
		result.CommonPrefixes = append(result.CommonPrefixes, p.Prefix)
	}
	return result, nil
}

// SignURL 调用 BOS SDK GeneratePresignedUrl 生成预签名 URL。
// method 支持 "GET" 和 "PUT"。
func (a *adapter) SignURL(_ context.Context, bucket, key, method string, expireSeconds int64) (string, error) {
	bucket = a.bucket(bucket)
	url := a.client.GeneratePresignedUrl(bucket, key, int(expireSeconds), method, nil, nil)
	return url, nil
}

// Copy 调用 BOS SDK CopyObject 实现服务端复制。
// 注意：BasicCopyObject 传 nil args 时 SDK 内部会 panic（bce-sdk-go bug），
// 因此显式传入空 CopyObjectArgs 规避该问题。
func (a *adapter) Copy(_ context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	srcBucket = a.bucket(srcBucket)
	dstBucket = a.bucket(dstBucket)

	if _, err := a.client.CopyObject(dstBucket, dstKey, srcBucket, srcKey, &bosapi.CopyObjectArgs{}); err != nil {
		return fmt.Errorf("oss/baidu: copy %s/%s → %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}
	return nil
}

// isNotFound 判断 BOS SDK 返回的错误是否为 404 Not Found。
func isNotFound(err error) bool {
	if svcErr, ok := err.(*bce.BceServiceError); ok {
		return svcErr.StatusCode == http.StatusNotFound
	}
	return false
}

// stripMetaPrefix 去除 BOS 自定义元数据前缀 "x-bce-meta-"。
func stripMetaPrefix(k string) string {
	return strings.TrimPrefix(k, "x-bce-meta-")
}
