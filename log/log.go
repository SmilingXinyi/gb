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

var (
	projectRoot     string
	activeSeqWriter io.Closer
)

func init() {
	workingDir, _ := os.Getwd()
	projectRoot = findProjectRoot(workingDir)

	// Set global CallerMarshalFunc to make the path relative to the project root
	zerolog.CallerMarshalFunc = func(pc uintptr, filePath string, lineNum int) string {
		if relative, err := filepath.Rel(projectRoot, filePath); err == nil {
			// If the path contains .., it means it's outside the project, so only show the base filename
			if !filepath.IsAbs(relative) && !strings.HasPrefix(relative, "..") {
				return relative + ":" + strconv.Itoa(lineNum)
			}
		}
		return filepath.Base(filePath) + ":" + strconv.Itoa(lineNum)
	}
}

func findProjectRoot(startDir string) string {
	currentDir := startDir
	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}
	return startDir
}

// Event is an alias for zerolog.Event for easier external use
type Event = zerolog.Event

// Setup initializes the global logger with the provided configuration
func Setup(config LogConfig) {
	var writersList []io.Writer

	// 1. Set log level
	logLevel := zerolog.InfoLevel
	if config.Console.Enabled {
		if config.Console.Level != "" {
			if parsedLevel, err := zerolog.ParseLevel(config.Console.Level); err == nil {
				logLevel = parsedLevel
			}
		} else {
			logLevel = zerolog.TraceLevel
		}
		writersList = append(writersList, writers.NewConsoleWriter())
	} else {
		writersList = append(writersList, os.Stderr)
	}
	zerolog.SetGlobalLevel(logLevel)

	// 2. Set file output
	fileWriter := writers.NewFileWriter(writers.FileWriterConfig{
		Filename:   config.File.Filename,
		MaxSize:    config.File.MaxSize,
		MaxBackups: config.File.MaxBackups,
		MaxAge:     config.File.MaxAge,
		Compress:   config.File.Compress,
	})
	if fileWriter != nil {
		writersList = append(writersList, fileWriter)
	}

	// 3. Set Seq output
	activeSeqWriter = nil
	if config.Seq.Enabled && config.Seq.Endpoint != "" {
		seqWriter := writers.NewSeqWriter(writers.SeqWriterConfig{
			Endpoint:      config.Seq.Endpoint,
			APIKey:        config.Seq.APIKey,
			Application:   config.Seq.Application,
			BatchSize:     config.Seq.BatchSize,
			FlushInterval: config.Seq.FlushInterval,
		})
		activeSeqWriter = seqWriter
		writersList = append(writersList, &zerolog.FilteredLevelWriter{
			Writer: zerolog.LevelWriterAdapter{Writer: seqWriter},
			Level:  resolveSeqLevel(config.Seq.Level),
		})
	}

	// 4. Merge outputs
	var multiWriter io.Writer
	if len(writersList) == 1 {
		multiWriter = writersList[0]
	} else {
		multiWriter = zerolog.MultiLevelWriter(writersList...)
	}

	log.Logger = zerolog.New(multiWriter).With().Timestamp().Caller().Logger()
}

// Shutdown flushes and closes optional asynchronous writers such as Seq.
func Shutdown() {
	if activeSeqWriter != nil {
		_ = activeSeqWriter.Close()
		activeSeqWriter = nil
	}
}

// resolveSeqLevel parses the configured Seq level and falls back to info.
func resolveSeqLevel(level string) zerolog.Level {
	if level == "" {
		return zerolog.InfoLevel
	}
	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return zerolog.InfoLevel
	}
	return parsedLevel
}

// Module returns a Logger with the specified module name
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
