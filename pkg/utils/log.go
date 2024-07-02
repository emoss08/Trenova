package utils

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func LogLevelFromString(s string) zerolog.Level {
	l, err := zerolog.ParseLevel(s)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse log level, defaulting to %s", zerolog.DebugLevel)
		return zerolog.DebugLevel
	}

	return l
}
