package testutils

import "github.com/trenova-app/transport/internal/pkg/logger"

func NewTestLogger() *logger.Logger {
	return logger.NewLogger(NewTestConfig())
}
