package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultLogEncoding = "json"
	defaultLogLevel    = zapcore.DebugLevel
	envLogEncoding     = "LOG_ENCODING"
	envLogLevel        = "LOG_LEVEL"
)

var (
	Log *zap.Logger
)

func init() {
	Init()
}

func Init() {
	enc := zap.NewProductionEncoderConfig()
	enc.EncodeTime = zapcore.RFC3339TimeEncoder

	config := zap.Config{
		Encoding:          logEncoding(),
		Level:             logLevel(),
		EncoderConfig:     enc,
		DisableCaller:     true,
		DisableStacktrace: true,
		OutputPaths:       []string{"stdout"},
	}

	var err error
	Log, err = config.Build()
	if err != nil {
		config.Encoding = defaultLogEncoding
		Log, err = config.Build()
	}
	if err != nil {
		Log = zap.NewNop()
	}
}

func logEncoding() string {
	encoding := strings.TrimSpace(os.Getenv(envLogEncoding))
	if encoding == "" {
		return defaultLogEncoding
	}
	return strings.ToLower(encoding)
}

func logLevel() zap.AtomicLevel {
	level := strings.TrimSpace(os.Getenv(envLogLevel))
	if level == "" {
		return zap.NewAtomicLevelAt(defaultLogLevel)
	}

	atomicLevel, err := zap.ParseAtomicLevel(strings.ToLower(level))
	if err != nil {
		return zap.NewAtomicLevelAt(defaultLogLevel)
	}
	return atomicLevel
}

func SyncLog() {
	Log.Sync()
}
