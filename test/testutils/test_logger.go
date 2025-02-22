package testutils

import (
	"testing"

	"github.com/emoss08/trenova/internal/pkg/logger"
)

func NewTestLogger(t *testing.T) *logger.Logger {
	return logger.NewLogger(NewTestConfig())
}
