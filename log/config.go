package log

// LogConfig 定义日志配置
type LogConfig struct {
	// Console 控制台输出配置
	Console ConsoleConfig
	// File 文件输出配置
	File FileConfig
}

// ConsoleConfig 控制台输出配置
type ConsoleConfig struct {
	// Enabled 是否开启控制台输出
	Enabled bool
	// Level 日志级别 (trace, debug, info, warn, error, fatal, panic)
	// 如果为空，Enabled 为 true 时默认为 trace，为 false 时默认为 info
	Level string
}

// FileConfig 文件输出配置
type FileConfig struct {
	// Filename 日志文件路径，如果为空则不输出到文件
	Filename string
	// MaxSize 每个日志文件的最大大小（MB）
	MaxSize int
	// MaxBackups 保留的旧日志文件最大数量
	MaxBackups int
	// MaxAge 保留的旧日志文件最大天数
	MaxAge int
	// Compress 是否压缩旧日志文件
	Compress bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() LogConfig {
	return LogConfig{
		Console: ConsoleConfig{
			Enabled: true,
			Level:   "trace",
		},
		File: FileConfig{
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
	}
}
