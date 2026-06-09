package oss

import "time"

// PutOptions 上传对象时的附加选项。
type PutOptions struct {
	// ContentType 指定对象的 MIME 类型（如 "image/png"）。
	// 留空则由各 provider SDK 自动推断或使用默认值。
	ContentType string

	// Metadata 自定义元数据键值对，会作为对象属性存储。
	// key 不含 provider 前缀（如 "x-bce-meta-"），适配器负责转换。
	Metadata map[string]string

	// StorageClass 存储类型（如 "STANDARD"、"COLD"、"ARCHIVE"）。
	// 留空则使用 bucket 默认存储类型。
	StorageClass string

	// ACL 对象访问控制（如 "private"、"public-read"）。
	// 留空则继承 bucket 默认 ACL。
	ACL string
}

// ObjectMeta 对象的元信息，不含对象内容。
type ObjectMeta struct {
	// Key 对象在 bucket 内的路径。
	Key string

	// Size 对象字节数。
	Size int64

	// ContentType 对象的 MIME 类型。
	ContentType string

	// ETag 对象内容的哈希标识（通常为 MD5 或 SHA256 的十六进制字符串）。
	ETag string

	// LastModified 对象最后修改时间。
	LastModified time.Time

	// Metadata 对象自定义元数据（适配器负责去除 provider 特定前缀后返回）。
	Metadata map[string]string

	// StorageClass 对象的存储类型。
	StorageClass string
}

// ObjectItem 是 List 结果中单个对象的摘要信息。
type ObjectItem struct {
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
	StorageClass string
}

// ListOptions List 操作的分页与过滤选项。
type ListOptions struct {
	// Delimiter 分组分隔符（通常为 "/"），用于模拟目录层级。
	// 设置后 ListResult.CommonPrefixes 会返回公共前缀（即"子目录"）。
	Delimiter string

	// MaxKeys 单次返回的最大对象数量，0 表示使用 provider 默认值（通常 1000）。
	MaxKeys int

	// ContinuationToken 分页续传令牌，来自上次 ListResult.NextToken。
	// 留空表示从头开始列举。
	ContinuationToken string
}

// ListResult List 操作的返回结果。
type ListResult struct {
	// Objects 本次列举到的对象列表。
	Objects []ObjectItem

	// CommonPrefixes 公共前缀列表（使用 Delimiter 时生效），可理解为"子目录"。
	CommonPrefixes []string

	// NextToken 下一页的续传令牌。为空字符串时表示已列举完毕。
	NextToken string

	// IsTruncated 是否还有更多对象未列举（true 表示需要继续翻页）。
	IsTruncated bool
}
