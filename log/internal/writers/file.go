package writers

import (
	"io"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// FileWriterConfig 文件输出配置接口
type FileWriterConfig struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// NewFileWriter 创建文件输出
func NewFileWriter(config FileWriterConfig) io.Writer {
	if config.Filename == "" {
		return nil
	}
	return &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}
}
