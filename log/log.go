package log

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SmilingXinyi/gb/log/internal/writers"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var projectRoot string

func init() {
	wd, _ := os.Getwd()
	projectRoot = findProjectRoot(wd)

	// 设置全局 CallerMarshalFunc，使路径相对于项目根目录
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		if rel, err := filepath.Rel(projectRoot, file); err == nil {
			// 如果路径包含 ..，说明在项目外部，则只显示文件名
			if !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..") {
				return rel + ":" + strconv.Itoa(line)
			}
		}
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}
}

func findProjectRoot(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return startDir
}

// Event 是 zerolog.Event 的别名，方便外部使用
type Event = zerolog.Event

// Setup 使用配置初始化全局日志
func Setup(config LogConfig) {
	var writersList []io.Writer

	// 1. 设置日志级别
	level := zerolog.InfoLevel
	if config.Console.Enabled {
		if config.Console.Level != "" {
			if l, err := zerolog.ParseLevel(config.Console.Level); err == nil {
				level = l
			}
		} else {
			level = zerolog.TraceLevel
		}
		writersList = append(writersList, writers.NewConsoleWriter())
	} else {
		writersList = append(writersList, os.Stderr)
	}
	zerolog.SetGlobalLevel(level)

	// 2. 设置文件输出
	fw := writers.NewFileWriter(writers.FileWriterConfig{
		Filename:   config.File.Filename,
		MaxSize:    config.File.MaxSize,
		MaxBackups: config.File.MaxBackups,
		MaxAge:     config.File.MaxAge,
		Compress:   config.File.Compress,
	})
	if fw != nil {
		writersList = append(writersList, fw)
	}

	// 3. 合并输出
	var multi io.Writer
	if len(writersList) == 1 {
		multi = writersList[0]
	} else {
		multi = zerolog.MultiLevelWriter(writersList...)
	}

	log.Logger = zerolog.New(multi).With().Timestamp().Caller().Logger()
}

// Module 返回一个带有指定模块名称的 Logger
func Module(name string) zerolog.Logger {
	return log.With().Str("module", name).Logger()
}

// Trace starts a new message with trace level.
func Trace() *Event {
	return log.Trace()
}

// Debug starts a new message with debug level.
func Debug() *Event {
	return log.Debug()
}

// Info starts a new message with info level.
func Info() *Event {
	return log.Info()
}

// Warn starts a new message with warn level.
func Warn() *Event {
	return log.Warn()
}

// Error starts a new message with error level.
func Error() *Event {
	return log.Error()
}

// Fatal starts a new message with fatal level.
func Fatal() *Event {
	return log.Fatal()
}

// Panic starts a new message with panic level.
func Panic() *Event {
	return log.Panic()
}

// Dict creates a new dictionary event.
func Dict() *Event {
	return zerolog.Dict()
}
