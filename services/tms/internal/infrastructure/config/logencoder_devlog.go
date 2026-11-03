//go:build devlog

package config

import (
	"os"

	"github.com/emoss08/trenova/internal/infrastructure/config/loggingconfig"
	"github.com/emoss08/trenova/pkg/prettylog"
	"github.com/mattn/go-isatty"
	"go.uber.org/zap/zapcore"
)

//nolint:unparam // the !devlog build returns ErrPrettyFormatNotBuilt through this signature
func newPrettyEncoder(cfg loggingconfig.Config) (zapcore.Encoder, error) {
	return prettylog.NewEncoder(prettylog.Options{Color: prettyColor(cfg)}), nil
}

func prettyColor(cfg loggingconfig.Config) bool {
	switch cfg.Color {
	case "always":
		return true
	case "never":
		return false
	}

	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	switch cfg.Output {
	case "stdout":
		return isatty.IsTerminal(os.Stdout.Fd())
	case "stderr":
		return isatty.IsTerminal(os.Stderr.Fd())
	default:
		return false
	}
}
