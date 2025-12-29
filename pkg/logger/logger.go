package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LoggerConfig struct {
	Verbose    bool
	JSONOutput bool
}

// ConfigLogger initializes the global logger with the provided configuration.
func ConfigLogger(cfg LoggerConfig) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if cfg.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if !cfg.JSONOutput {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}
