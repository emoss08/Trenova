package temporaljobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TemporalClientParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

func NewTemporalClient(p TemporalClientParams) client.Client {
	log := p.Logger.Named("temporal-client")
	cfg := p.Config.Temporal

	clientOptions := client.Options{
		HostPort: cfg.HostPort,
	}

	if cfg.Security.EnableEncryption || cfg.Security.EnableCompression {
		log.Info("Configuring Temporal data converter with security features",
			zap.Bool("encryption", cfg.Security.EnableEncryption),
			zap.Bool("compression", cfg.Security.EnableCompression),
		)

		dataConverterOptions := temporaltype.DataConverterOptions{
			EnableEncryption:     cfg.Security.EnableEncryption,
			EncryptionKeyID:      cfg.Security.EncryptionKeyID,
			EnableCompression:    cfg.Security.EnableCompression,
			CompressionThreshold: cfg.Security.CompressionThreshold,
		}

		clientOptions.DataConverter = temporaltype.NewEncryptionDataConverter(dataConverterOptions)

		if cfg.Security.EnableEncryption {
			log.Warn(
				"Encryption enabled - ensure TEMPORAL_ENCRYPTION_KEY environment variable is set",
				zap.String("keyID", cfg.Security.EncryptionKeyID),
			)
		}
	}

	c, err := client.Dial(clientOptions)
	if err != nil {
		log.Fatal("failed to dial temporal client", zap.Error(err))
	}

	log.Info("Temporal client dialed successfully",
		zap.String("hostPort", cfg.HostPort),
		zap.Bool("encryptionEnabled", cfg.Security.EnableEncryption),
		zap.Bool("compressionEnabled", cfg.Security.EnableCompression),
	)

	return c
}
