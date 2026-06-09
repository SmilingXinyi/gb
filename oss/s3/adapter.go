// Package s3 是 AWS S3 及 S3 兼容存储（MinIO、Cloudflare R2、七牛云等）的 Storage 适配器。
// 通过在 init() 中调用 oss.Register(oss.ProviderS3, newAdapter) 完成自动注册。
//
// 依赖 SDK：github.com/aws/aws-sdk-go-v2
// 使用前需在 go.mod 中引入以下依赖：
//
//	github.com/aws/aws-sdk-go-v2
//	github.com/aws/aws-sdk-go-v2/config
//	github.com/aws/aws-sdk-go-v2/credentials
//	github.com/aws/aws-sdk-go-v2/service/s3
//
// 配置字段映射：
//
//	Config.AccessKey  → AWS Access Key ID
//	Config.SecretKey  → AWS Secret Access Key
//	Config.Token      → AWS Session Token（临时凭证时使用）
//	Config.Region     → AWS Region（如 "us-east-1"）
//	Config.Endpoint   → 自定义 Endpoint URL（MinIO、R2 等 S3 兼容服务必填）
//	Config.Bucket     → 默认 bucket（可选）
package s3

import (
	"context"
	"io"

	"github.com/SmilingXinyi/gb/oss"
)

func init() {
	oss.Register(oss.ProviderS3, newAdapter)
}

// adapter 实现 oss.Storage 接口，封装 AWS S3 SDK v2 客户端。
type adapter struct {
	// TODO: 持有 *s3.Client 实例及默认 bucket 配置
	cfg oss.Config
}

// newAdapter 根据 Config 创建并初始化 S3 adapter。
// 负责：
//  1. 校验必填字段（AccessKey、SecretKey、Region）
//  2. 使用 credentials.NewStaticCredentialsProvider 构建静态凭证
//  3. 若 cfg.Endpoint 非空，通过 config.WithEndpointResolverWithOptions 注入自定义 Endpoint
//     并设置 s3.WithUsePathStyle(true)（MinIO 等需要 path-style）
//  4. 调用 s3.NewFromConfig 初始化客户端
func newAdapter(cfg oss.Config) (oss.Storage, error) {
	// TODO
	return &adapter{cfg: cfg}, nil
}

// Put 调用 S3 SDK 的 PutObject 上传对象。
// 需将 PutOptions 中的 ContentType、Metadata、StorageClass、ACL 转换为 s3.PutObjectInput 字段。
// size >= 0 时设置 ContentLength；size == -1 时不设置（部分 S3 兼容服务要求必须提供）。
func (a *adapter) Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts *oss.PutOptions) error {
	// TODO
	return nil
}

// Get 调用 S3 SDK 的 GetObject 下载对象，返回响应体 ReadCloser。
// 需将 NoSuchKey 错误（smithy APIError）转换为 oss.ErrObjectNotFound。
func (a *adapter) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	// TODO
	return nil, nil
}

// Delete 调用 S3 SDK 的 DeleteObject 删除对象。
func (a *adapter) Delete(ctx context.Context, bucket, key string) error {
	// TODO
	return nil
}

// Stat 调用 S3 SDK 的 HeadObject 获取对象元信息（HTTP HEAD 请求）。
// 需将响应中的 ContentType、ETag、ContentLength、LastModified、Metadata 映射到 ObjectMeta。
func (a *adapter) Stat(ctx context.Context, bucket, key string) (*oss.ObjectMeta, error) {
	// TODO
	return nil, nil
}

// List 调用 S3 SDK 的 ListObjectsV2 列举对象。
// 需处理分页（ContinuationToken）及 CommonPrefixes（使用 Delimiter 时）。
func (a *adapter) List(ctx context.Context, bucket, prefix string, opts *oss.ListOptions) (*oss.ListResult, error) {
	// TODO
	return nil, nil
}

// SignURL 调用 S3 SDK 的 presign.Client.PresignGetObject / PresignPutObject 生成预签名 URL。
// method 支持 "GET" 和 "PUT"，expireSeconds 转换为 time.Duration 传入。
func (a *adapter) SignURL(ctx context.Context, bucket, key, method string, expireSeconds int64) (string, error) {
	// TODO
	return "", nil
}

// Copy 调用 S3 SDK 的 CopyObject 实现服务端复制。
// 需将源对象路径格式化为 "{srcBucket}/{srcKey}" 作为 CopySource。
func (a *adapter) Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	// TODO
	return nil
}
