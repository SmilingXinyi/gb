package oss

import "fmt"

// ErrProviderNotSupported 使用了未注册的 provider 时返回此错误。
type ErrProviderNotSupported struct {
	Provider Provider
}

func (e *ErrProviderNotSupported) Error() string {
	return fmt.Sprintf("oss: provider %q is not supported", e.Provider)
}

// ErrObjectNotFound 对象不存在时返回此错误。
// 适配器应将 provider SDK 的 404/NoSuchKey 等错误统一转换为此类型。
type ErrObjectNotFound struct {
	Bucket string
	Key    string
}

func (e *ErrObjectNotFound) Error() string {
	return fmt.Sprintf("oss: object not found: %s/%s", e.Bucket, e.Key)
}

// ErrBucketNotFound bucket 不存在时返回此错误。
type ErrBucketNotFound struct {
	Bucket string
}

func (e *ErrBucketNotFound) Error() string {
	return fmt.Sprintf("oss: bucket not found: %s", e.Bucket)
}

// ErrInvalidConfig 配置参数不合法时返回此错误。
type ErrInvalidConfig struct {
	Field   string
	Message string
}

func (e *ErrInvalidConfig) Error() string {
	return fmt.Sprintf("oss: invalid config field %q: %s", e.Field, e.Message)
}
