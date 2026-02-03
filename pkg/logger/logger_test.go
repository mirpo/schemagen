package logger

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestConfigLogger_DefaultLevel(t *testing.T) {
	// Default should be Info level
	ConfigLogger(LoggerConfig{Verbose: false, JSONOutput: true})
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("expected InfoLevel, got %v", zerolog.GlobalLevel())
	}
}

func TestConfigLogger_VerboseLevel(t *testing.T) {
	// Verbose should set Debug level
	ConfigLogger(LoggerConfig{Verbose: true, JSONOutput: true})
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("expected DebugLevel, got %v", zerolog.GlobalLevel())
	}

	// Reset to default
	ConfigLogger(LoggerConfig{Verbose: false, JSONOutput: true})
}

func TestConfigLogger_ConsoleOutput(t *testing.T) {
	// Should not panic with console output
	ConfigLogger(LoggerConfig{Verbose: false, JSONOutput: false})

	// Reset to default
	ConfigLogger(LoggerConfig{Verbose: false, JSONOutput: true})
}
