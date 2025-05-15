package db

import "go.uber.org/zap"

var logger *zap.Logger

// InitializeLogger sets the logger for the db package.
func InitializeLogger(l *zap.Logger) {
	logger = l
}
