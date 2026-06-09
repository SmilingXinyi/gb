package oss

// New 根据 provider 类型和配置创建对应的 Storage 实例。
// 这是整个 oss 包的统一入口，上层业务代码只需调用此函数。
//
// provider 参数支持 oss.Provider 类型或 string 类型。
//
// 示例：
//
//	// 使用 Provider 类型
//	client, err := oss.New(oss.ProviderBaidu, oss.Config{...})
//
//	// 使用 string 类型
//	client, err := oss.New("baidu", oss.Config{...})
func New(p any, cfg Config) (Storage, error) {
	var provider Provider
	switch v := p.(type) {
	case Provider:
		provider = v
	case string:
		provider = Provider(v)
	default:
		return nil, &ErrProviderNotSupported{}
	}

	cfg.Provider = provider
	factory, ok := adapterRegistry[provider]
	if !ok {
		return nil, &ErrProviderNotSupported{Provider: provider}
	}
	return factory(cfg)
}

// adapterFactory 是各 provider adapter 的构造函数签名。
type adapterFactory func(cfg Config) (Storage, error)

// adapterRegistry 存储已注册的 provider → factory 映射。
// 各 provider 子包在 init() 中调用 Register 完成注册。
var adapterRegistry = map[Provider]adapterFactory{}

// Register 注册一个 provider 的 adapter 工厂函数。
// 通常在各 provider 子包的 init() 中调用，实现自动注册。
// 若同一 provider 重复注册，后注册的会覆盖前者（panic 式检查可按需开启）。
func Register(provider Provider, factory adapterFactory) {
	adapterRegistry[provider] = factory
}
