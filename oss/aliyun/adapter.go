// Package aliyun 是阿里云 OSS（Object Storage Service）的 Storage 适配器。
// 通过在 init() 中调用 oss.Register(oss.ProviderAliyun, newAdapter) 完成自动注册。
//
// 依赖 SDK：github.com/aliyun/aliyun-oss-go-sdk/oss
// 使用前需在 go.mod 中引入该依赖。
//
// 配置字段映射：
//
//	Config.AccessKey  → AccessKeyID
//	Config.SecretKey  → AccessKeySecret
//	Config.Token      → STS SecurityToken（临时凭证时使用）
//	Config.Endpoint   → OSS Endpoint（如 "https://oss-cn-hangzhou.aliyuncs.com"）
//	Config.Region     → 若 Endpoint 为空，用于推导 Endpoint
//	Config.Bucket     → 默认 bucket（可选）
package aliyun

import (
	"context"
	"io"

	"github.com/SmilingXinyi/gb/oss"
)

func init() {
	oss.Register(oss.ProviderAliyun, newAdapter)
}

// adapter 实现 oss.Storage 接口，封装阿里云 OSS SDK 客户端。
type adapter struct {
	// TODO: 持有阿里云 OSS SDK client 实例及默认 bucket 配置
	cfg oss.Config
}

// newAdapter 根据 Config 创建并初始化阿里云 OSS adapter。
// 负责：
//  1. 校验必填字段（AccessKey、SecretKey、Endpoint 或 Region）
//  2. 若 Endpoint 为空，根据 Region 推导（"oss-{Region}.aliyuncs.com"）
//  3. 若 Token 非空，使用 STS 临时凭证初始化客户端
//  4. 初始化阿里云 OSS SDK client
func newAdapter(cfg oss.Config) (oss.Storage, error) {
	// TODO
	return &adapter{cfg: cfg}, nil
}

// Put 调用阿里云 OSS SDK 的 bucket.PutObject 上传对象。
// 需将 PutOptions 中的 ContentType、Metadata、StorageClass、ACL 转换为 oss.Option 列表。
func (a *adapter) Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts *oss.PutOptions) error {
	// TODO
	return nil
}

// Get 调用阿里云 OSS SDK 的 bucket.GetObject 下载对象。
// 需将 NoSuchKey 错误转换为 oss.ErrObjectNotFound。
func (a *adapter) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	// TODO
	return nil, nil
}

// Delete 调用阿里云 OSS SDK 的 bucket.DeleteObject 删除对象。
func (a *adapter) Delete(ctx context.Context, bucket, key string) error {
	// TODO
	return nil
}

// Stat 调用阿里云 OSS SDK 的 bucket.GetObjectMeta 获取对象元信息。
// 需将响应头（Content-Type、ETag、Content-Length、Last-Modified、x-oss-meta-* 等）映射到 ObjectMeta。
func (a *adapter) Stat(ctx context.Context, bucket, key string) (*oss.ObjectMeta, error) {
	// TODO
	return nil, nil
}

// List 调用阿里云 OSS SDK 的 bucket.ListObjectsV2 列举对象。
// 需处理分页（ContinuationToken）及 CommonPrefixes（模拟目录）。
func (a *adapter) List(ctx context.Context, bucket, prefix string, opts *oss.ListOptions) (*oss.ListResult, error) {
	// TODO
	return nil, nil
}

// SignURL 调用阿里云 OSS SDK 的 bucket.SignURL 生成预签名 URL。
// method 支持 "GET" 和 "PUT"，expireSeconds 为有效期秒数。
func (a *adapter) SignURL(ctx context.Context, bucket, key, method string, expireSeconds int64) (string, error) {
	// TODO
	return "", nil
}

// Copy 调用阿里云 OSS SDK 的 bucket.CopyObject 实现服务端复制。
// 跨 bucket 复制时需通过 oss.CopySourceBucketName 指定源 bucket。
func (a *adapter) Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	// TODO
	return nil
}
