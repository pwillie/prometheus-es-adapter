package logger

import (
	"go.uber.org/zap"
)

func NewLogger(debug bool) *zap.Logger {
	var logger *zap.Logger
	if debug {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync() // flushes buffer, if any
	return logger
}
