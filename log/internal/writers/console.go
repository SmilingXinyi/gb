package writers

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
)

// LevelColors 定义日志级别到颜色的映射
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

// NewConsoleWriter 创建控制台输出
func NewConsoleWriter() io.Writer {
	return zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05.000",
		FormatLevel: func(i interface{}) string {
			var l string
			if ll, ok := i.(string); ok {
				l = strings.ToUpper(ll)
			}
			level, _ := zerolog.ParseLevel(strings.ToLower(l))
			c, ok := LevelColors[level]
			if !ok {
				c = color.New(color.FgWhite)
			}
			return colorBold.Sprint(c.Sprintf("%-5s", l))
		},
		FormatTimestamp: func(i interface{}) string {
			return colorDim.Sprint(i)
		},
		FormatCaller: func(i interface{}) string {
			return colorDim.Sprint(i)
		},
		FormatMessage: func(i interface{}) string {
			return fmt.Sprintf("%v", i)
		},
		FormatFieldName: func(i interface{}) string {
			return colorDim.Sprintf("%s=", i)
		},
		FormatFieldValue: func(i interface{}) string {
			return fmt.Sprintf("%v", i)
		},
	}
}
