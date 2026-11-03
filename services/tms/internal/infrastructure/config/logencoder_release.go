//go:build !devlog

package config

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/loggingconfig"
	"go.uber.org/zap/zapcore"
)

func newPrettyEncoder(loggingconfig.Config) (zapcore.Encoder, error) {
	return nil, loggingconfig.ErrPrettyFormatNotBuilt
}
