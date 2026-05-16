package main

import (
	"github.com/SmilingXinyi/gb/log"
)

func main() {
	// 1. Get default configuration and modify it
	config := log.DefaultConfig()
	config.File.Filename = "app.log" // Output to both console and app.log
	config.Console.Enabled = true
	config.Console.Level = "trace"

	// 2. Initialize
	log.Setup(config)

	// 3. Demonstrate all common log levels
	log.Trace().Msg("This is a trace log")
	log.Debug().Msg("This is a debug log")
	log.Info().Msg("This is an info log")
	log.Warn().Msg("This is a warn log")
	log.Error().Msg("This is an error log")

	// 4. Log with fields
	log.Info().
		Str("version", "1.0.0").
		Int("port", 8080).
		Msg("Project startup details")

	// 5. Modular logging
	authLogger := log.Module("auth")
	authLogger.Info().Msg("User authentication module is ready")

	// 6. Nested dictionary logging
	log.Info().
		Dict("database", log.Dict().
			Str("host", "localhost").
			Int("conns", 10),
		).Msg("Database connection pool status")
}
