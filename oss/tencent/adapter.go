// Package tencent 是腾讯云 COS（Cloud Object Storage）的 Storage 适配器。
// 通过在 init() 中调用 oss.Register(oss.ProviderTencent, newAdapter) 完成自动注册。
//
// 依赖 SDK：github.com/tencentyun/cos-go-sdk-v5
// 使用前需在 go.mod 中引入该依赖。
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
	"io"

	"github.com/SmilingXinyi/gb/oss"
)

func init() {
	oss.Register(oss.ProviderTencent, newAdapter)
}

// adapter 实现 oss.Storage 接口，封装腾讯云 COS SDK 客户端。
type adapter struct {
	// TODO: 持有 COS SDK cos.Client 实例及默认 bucket、region 配置
	cfg oss.Config
}

// newAdapter 根据 Config 创建并初始化腾讯云 COS adapter。
// 负责：
//  1. 校验必填字段（AccessKey、SecretKey、Region）
//  2. 根据 bucket + region 构造 BucketURL（或直接使用 cfg.Endpoint）
//  3. 若 Token 非空，注入 SessionToken 到 Transport
//  4. 初始化 cos.NewClient
func newAdapter(cfg oss.Config) (oss.Storage, error) {
	// TODO
	return &adapter{cfg: cfg}, nil
}

// bucketURL 根据 bucket 名称和 region 生成腾讯云 COS bucket URL。
// 格式：https://{bucket}.cos.{region}.myqcloud.com
func bucketURL(bucket, region string) string {
	// TODO
	return ""
}

// Put 调用 COS SDK 的 Object.Put 上传对象。
// 需将 PutOptions 中的 ContentType、Metadata、StorageClass、ACL 转换为 cos.ObjectPutOptions。
func (a *adapter) Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts *oss.PutOptions) error {
	// TODO
	return nil
}

// Get 调用 COS SDK 的 Object.Get 下载对象。
// 需将 NoSuchKey 错误转换为 oss.ErrObjectNotFound。
func (a *adapter) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	// TODO
	return nil, nil
}

// Delete 调用 COS SDK 的 Object.Delete 删除对象。
func (a *adapter) Delete(ctx context.Context, bucket, key string) error {
	// TODO
	return nil
}

// Stat 调用 COS SDK 的 Object.Head 获取对象元信息（HTTP HEAD 请求）。
// 需将响应头（Content-Type、ETag、Content-Length、Last-Modified、x-cos-meta-* 等）映射到 ObjectMeta。
func (a *adapter) Stat(ctx context.Context, bucket, key string) (*oss.ObjectMeta, error) {
	// TODO
	return nil, nil
}

// List 调用 COS SDK 的 Bucket.Get（ListObjectsV2）列举对象。
// 需处理分页（ContinuationToken）及 CommonPrefixes。
func (a *adapter) List(ctx context.Context, bucket, prefix string, opts *oss.ListOptions) (*oss.ListResult, error) {
	// TODO
	return nil, nil
}

// SignURL 调用 COS SDK 的 Object.GetPresignedURL 生成预签名 URL。
// method 支持 "GET" 和 "PUT"，expireSeconds 转换为 time.Duration 传入。
func (a *adapter) SignURL(ctx context.Context, bucket, key, method string, expireSeconds int64) (string, error) {
	// TODO
	return "", nil
}

// Copy 调用 COS SDK 的 Object.Copy 实现服务端复制。
// 需构造源对象 URL（格式 "{srcBucket}.cos.{region}.myqcloud.com/{srcKey}"）。
func (a *adapter) Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	// TODO
	return nil
}
