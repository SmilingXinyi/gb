package main

import (
	"time"

	"github.com/SmilingXinyi/gb/log"
)

func main() {
	config := log.DefaultConfig()
	config.Console.Enabled = true
	config.Console.Level = "trace"
	config.Seq = log.SeqConfig{
		Enabled:       true,
		Endpoint:      "http://localhost:5342/ingest/clef",
		Level:         "trace",
		Application:   "demo-app",
		BatchSize:     10,
		FlushInterval: 2 * time.Second,
	}

	log.Setup(config)
	defer log.Shutdown()

	log.Info().Str("version", "1.0.0").Msg("service started")
	log.Trace().Str("step", "init").Msg("initialization complete")

	authLogger := log.Module("auth")
	authLogger.Warn().Str("user", "alice").Msg("password expiring soon")
}
