package logging

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		zapLevel = zap.DebugLevel
	case "warn", "warning":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)
	return cfg.Build()
}
