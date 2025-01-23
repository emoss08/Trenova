package testutils

import "github.com/emoss08/trenova/internal/pkg/logger"

func NewTestLogger() *logger.Logger {
	return logger.NewLogger(NewTestConfig())
}
