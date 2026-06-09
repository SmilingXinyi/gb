package oss

// Config 是创建 Storage 所需的通用配置。
// 不同 provider 对字段的含义略有差异，详见各 adapter 文档。
type Config struct {
	// Provider 指定使用哪个存储服务商。
	// 也可通过 New(provider, cfg) 的第一个参数指定，两者一致即可。
	Provider Provider

	// Endpoint 服务接入点 URL（可选）。
	// 留空则使用 provider 默认端点；S3 兼容存储通常需要显式指定。
	Endpoint string

	// Region 存储区域（如 "cn-north-1"、"ap-southeast-1"）。
	Region string

	// AccessKey / SecretKey 鉴权凭证。
	// 百度云对应 AK/SK，阿里云对应 AccessKeyID/AccessKeySecret，以此类推。
	AccessKey string
	SecretKey string

	// Token 临时安全令牌（STS Token），使用临时凭证时填写。
	Token string

	// Bucket 默认操作的存储桶名称（可选）。
	// 若设置，调用方可省略 bucket 参数；不设置则每次调用必须显式传入。
	Bucket string
}
