package main

import (
	"github.com/SmilingXinyi/gb/log"
)

func main() {
	// 1. 获取默认配置并修改
	config := log.DefaultConfig()
	config.File.Filename = "app.log" // 同时输出到控制台和 app.log
	config.Console.Enabled = true
	config.Console.Level = "trace"

	// 2. 初始化
	log.Setup(config)

	// 3. 演示所有常用日志级别
	log.Trace().Msg("这是一条追踪日志 (Trace)")
	log.Debug().Msg("这是一条调试日志 (Debug)")
	log.Info().Msg("这是一条普通日志 (Info)")
	log.Warn().Msg("这是一条警告日志 (Warn)")
	log.Error().Msg("这是一条错误日志 (Error)")

	// 4. 带有字段的日志
	log.Info().
		Str("version", "1.0.0").
		Int("port", 8080).
		Msg("项目启动详情")

	// 5. 模块化日志
	authLogger := log.Module("auth")
	authLogger.Info().Msg("用户认证模块已就绪")

	// 6. 嵌套字典日志
	log.Info().
		Dict("database", log.Dict().
			Str("host", "localhost").
			Int("conns", 10),
		).Msg("数据库连接池状态")
}
