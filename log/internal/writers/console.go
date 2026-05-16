package writers

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
)

// LevelColors defines the mapping from log levels to colors
var LevelColors = map[zerolog.Level]*color.Color{
	zerolog.TraceLevel: color.New(color.FgHiBlack),   // Gray
	zerolog.DebugLevel: color.New(color.FgCyan),      // Cyan
	zerolog.InfoLevel:  color.New(color.FgGreen),     // Green
	zerolog.WarnLevel:  color.New(color.FgYellow),    // Yellow
	zerolog.ErrorLevel: color.New(color.FgRed),       // Red
	zerolog.FatalLevel: color.New(color.FgMagenta),   // Magenta
	zerolog.PanicLevel: color.New(color.FgHiMagenta), // Bright Magenta
}

var (
	colorBold = color.New(color.Bold)
	colorDim  = color.New(color.Faint)
)

// NewConsoleWriter creates a new console writer with formatted output
func NewConsoleWriter() io.Writer {
	return zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05.000",
		FormatLevel: func(levelValue interface{}) string {
			var levelStr string
			if str, ok := levelValue.(string); ok {
				levelStr = strings.ToUpper(str)
			}
			parsedLevel, _ := zerolog.ParseLevel(strings.ToLower(levelStr))
			levelColor, ok := LevelColors[parsedLevel]
			if !ok {
				levelColor = color.New(color.FgWhite)
			}
			return colorBold.Sprint(levelColor.Sprintf("%-5s", levelStr))
		},
		FormatTimestamp: func(timestamp interface{}) string {
			return colorDim.Sprint(timestamp)
		},
		FormatCaller: func(caller interface{}) string {
			return colorDim.Sprint(caller)
		},
		FormatMessage: func(message interface{}) string {
			return fmt.Sprintf("%v", message)
		},
		FormatFieldName: func(fieldName interface{}) string {
			return colorDim.Sprintf("%s=", fieldName)
		},
		FormatFieldValue: func(fieldValue interface{}) string {
			return fmt.Sprintf("%v", fieldValue)
		},
	}
}
