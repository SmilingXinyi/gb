package log

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestSetup(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		Setup(config)
		if zerolog.GlobalLevel() != zerolog.TraceLevel {
			t.Errorf("Expected global level to be TraceLevel, got %v", zerolog.GlobalLevel())
		}
	})

	t.Run("CustomLevel", func(t *testing.T) {
		config := DefaultConfig()
		config.Console.Level = "error"
		Setup(config)
		if zerolog.GlobalLevel() != zerolog.ErrorLevel {
			t.Errorf("Expected global level to be ErrorLevel, got %v", zerolog.GlobalLevel())
		}
	})

	t.Run("InvalidLevel", func(t *testing.T) {
		config := DefaultConfig()
		config.Console.Level = "invalid"
		Setup(config)
		// Should fallback to InfoLevel if Console.Enabled is true but level is invalid
		//
		// According to log.go:
		// level := zerolog.InfoLevel
		// if config.Console.Enabled {
		//    if config.Console.Level != "" {
		//        if l, err := zerolog.ParseLevel(config.Console.Level); err == nil { level = l }
		//    } else { level = zerolog.TraceLevel }
		// }
		// So if Level is "invalid", it stays InfoLevel.
		if zerolog.GlobalLevel() != zerolog.InfoLevel {
			t.Errorf("Expected global level to be InfoLevel for invalid level, got %v", zerolog.GlobalLevel())
		}
	})

	t.Run("FileLogging", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test.log")
		config := DefaultConfig()
		config.File.Filename = tmpFile
		Setup(config)

		Info().Msg("test message")

		if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
			t.Errorf("Log file %s was not created", tmpFile)
		}
	})
}

func TestCallerMarshalFunc(t *testing.T) {
	// projectRoot is set in init(). We can test the global CallerMarshalFunc directly.
	if zerolog.CallerMarshalFunc == nil {
		t.Fatal("zerolog.CallerMarshalFunc is not set")
	}

	tests := []struct {
		name     string
		file     string
		line     int
		expected string
	}{
		{
			name:     "InternalPath",
			file:     filepath.Join(projectRoot, "log/log.go"),
			line:     10,
			expected: "log/log.go:10",
		},
		{
			name:     "ExternalPath",
			file:     "/tmp/external.go",
			line:     20,
			expected: "external.go:20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := zerolog.CallerMarshalFunc(0, tt.file, tt.line)
			if !strings.HasSuffix(got, tt.expected) {
				t.Errorf("CallerMarshalFunc() = %v, want suffix %v", got, tt.expected)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	// Capture output to verify levels
	var buf bytes.Buffer
	oldLogger := log.Logger
	defer func() { log.Logger = oldLogger }()

	log.Logger = zerolog.New(&buf)

	t.Run("Trace", func(t *testing.T) {
		buf.Reset()
		Trace().Msg("trace")
		if !strings.Contains(buf.String(), `"level":"trace"`) {
			t.Errorf("Trace() output missing level: %s", buf.String())
		}
	})

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		Debug().Msg("debug")
		if !strings.Contains(buf.String(), `"level":"debug"`) {
			t.Errorf("Debug() output missing level: %s", buf.String())
		}
	})

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		Info().Msg("info")
		if !strings.Contains(buf.String(), `"level":"info"`) {
			t.Errorf("Info() output missing level: %s", buf.String())
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		Warn().Msg("warn")
		if !strings.Contains(buf.String(), `"level":"warn"`) {
			t.Errorf("Warn() output missing level: %s", buf.String())
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		Error().Msg("error")
		if !strings.Contains(buf.String(), `"level":"error"`) {
			t.Errorf("Error() output missing level: %s", buf.String())
		}
	})

	t.Run("Dict", func(t *testing.T) {
		buf.Reset()
		Info().Dict("details", Dict().Str("key", "value")).Msg("msg")
		if !strings.Contains(buf.String(), `"details":{"key":"value"}`) {
			t.Errorf("Dict() output incorrect: %s", buf.String())
		}
	})

	t.Run("Panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Panic() did not panic")
			}
		}()
		Panic().Msg("panic message")
	})
}

func TestModule(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := log.Logger
	defer func() { log.Logger = oldLogger }()

	log.Logger = zerolog.New(&buf)

	logger := Module("test-module")
	logger.Info().Msg("hello")

	if !strings.Contains(buf.String(), `"module":"test-module"`) {
		t.Errorf("Module() output missing module field: %s", buf.String())
	}
}
