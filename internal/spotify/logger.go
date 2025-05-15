package spotify

import "go.uber.org/zap"

var logger *zap.Logger

// InitializeLogger sets the logger for the spotify package.
func InitializeLogger(l *zap.Logger) {
	logger = l
}
