package temporaljobs

import (
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type TemporalClientParams struct {
	fx.In

	Config *config.Manager
	Logger *logger.Logger
}

func NewTemporalClient(p TemporalClientParams) client.Client {
	cfg := p.Config.Temporal()
	log := p.Logger.With().
		Str("component", "temporal-client").
		Logger()

	// Build client options
	clientOptions := client.Options{
		HostPort: cfg.HostPort,
	}

	// Configure data converter if security features are enabled
	if cfg.Security.EnableEncryption || cfg.Security.EnableCompression {
		log.Info().
			Bool("encryption", cfg.Security.EnableEncryption).
			Bool("compression", cfg.Security.EnableCompression).
			Msg("Configuring Temporal data converter with security features")

		dataConverterOptions := temporaltype.DataConverterOptions{
			EnableEncryption:     cfg.Security.EnableEncryption,
			EncryptionKeyID:      cfg.Security.EncryptionKeyID,
			EnableCompression:    cfg.Security.EnableCompression,
			CompressionThreshold: cfg.Security.CompressionThreshold,
		}

		clientOptions.DataConverter = temporaltype.NewEncryptionDataConverter(dataConverterOptions)

		if cfg.Security.EnableEncryption {
			log.Warn().
				Str("keyID", cfg.Security.EncryptionKeyID).
				Msg("Encryption enabled - ensure TEMPORAL_ENCRYPTION_KEY environment variable is set")
		}
	}

	c, err := client.Dial(clientOptions)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial temporal client")
	}

	log.Info().
		Str("hostPort", cfg.HostPort).
		Bool("encryptionEnabled", cfg.Security.EnableEncryption).
		Bool("compressionEnabled", cfg.Security.EnableCompression).
		Msg("temporal client dialed successfully")

	return c
}
