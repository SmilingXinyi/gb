// Package oss 提供统一的对象存储抽象层。
// 支持多个云存储 provider（百度云、阿里云、腾讯云、AWS S3），
// 通过 Storage 接口屏蔽底层 SDK 差异，上层业务代码无需感知具体 provider。
//
// 基本用法：
//
//	client, err := oss.New(oss.ProviderBaidu, oss.Config{...})
//	if err != nil { ... }
//	err = client.Put(ctx, "my-bucket", "path/to/file.txt", reader, opts)
package oss

import (
	"context"
	"io"
)

// Provider 标识对象存储服务商。
type Provider string

const (
	ProviderBaidu   Provider = "baidu"   // 百度云 BOS
	ProviderAliyun  Provider = "aliyun"  // 阿里云 OSS
	ProviderTencent Provider = "tencent" // 腾讯云 COS
	ProviderS3      Provider = "s3"      // AWS S3 或其他 S3 兼容存储
)

// Storage 是对象存储服务的统一操作接口。
// 所有 provider 的 adapter 必须实现该接口。
type Storage interface {
	// Put 上传对象到指定 bucket 的 key 路径。
	// size 为对象字节数（-1 表示未知，由实现自行处理）。
	// opts 可携带 ContentType、元数据等附加选项。
	Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts *PutOptions) error

	// Get 下载指定 bucket/key 的对象，返回可读流。
	// 调用方负责关闭返回的 ReadCloser。
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)

	// Delete 删除指定 bucket/key 的对象。
	Delete(ctx context.Context, bucket, key string) error

	// Stat 获取对象元信息（大小、MIME、修改时间等），不下载内容。
	Stat(ctx context.Context, bucket, key string) (*ObjectMeta, error)

	// List 列举 bucket 内指定前缀下的对象。
	// opts 控制分页、分隔符等行为。
	List(ctx context.Context, bucket, prefix string, opts *ListOptions) (*ListResult, error)

	// SignURL 生成带时效的预签名访问 URL（用于临时授权下载/上传）。
	// method 为 HTTP 方法（"GET" 或 "PUT"），expire 为有效时长（秒）。
	SignURL(ctx context.Context, bucket, key, method string, expireSeconds int64) (string, error)

	// Copy 在同一 provider 内服务端复制对象，避免客户端下载再上传。
	Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error
}
