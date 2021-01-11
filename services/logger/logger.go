package logger

import "go.uber.org/zap"

// New zap logger
func New() *zap.Logger {
	logger, _ := zap.NewProduction()

	return logger
}
