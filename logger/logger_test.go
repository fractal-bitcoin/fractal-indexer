package logger

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestLogEncodingDefault(t *testing.T) {
	t.Setenv(envLogEncoding, "")

	if got := logEncoding(); got != defaultLogEncoding {
		t.Fatalf("logEncoding() = %q, want %q", got, defaultLogEncoding)
	}
}

func TestLogEncodingFromEnv(t *testing.T) {
	t.Setenv(envLogEncoding, " CONSOLE ")

	if got := logEncoding(); got != "console" {
		t.Fatalf("logEncoding() = %q, want %q", got, "console")
	}
}

func TestLogLevelDefault(t *testing.T) {
	t.Setenv(envLogLevel, "")

	if got := logLevel().Level(); got != defaultLogLevel {
		t.Fatalf("logLevel() = %s, want %s", got, defaultLogLevel)
	}
}

func TestLogLevelFromEnv(t *testing.T) {
	t.Setenv(envLogLevel, " WARN ")

	if got := logLevel().Level(); got != zapcore.WarnLevel {
		t.Fatalf("logLevel() = %s, want %s", got, zapcore.WarnLevel)
	}
}

func TestLogLevelInvalidUsesDefault(t *testing.T) {
	t.Setenv(envLogLevel, "invalid")

	if got := logLevel().Level(); got != defaultLogLevel {
		t.Fatalf("logLevel() = %s, want %s", got, defaultLogLevel)
	}
}
