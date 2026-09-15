package temporaljobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/temporaljobs/connection"
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
	LC     fx.Lifecycle
}

type TemporalClientResult struct {
	fx.Out

	Client client.Client
}

func NewTemporalClient(p TemporalClientParams) (TemporalClientResult, error) {
	log := p.Logger.Named("temporal-client")
	cfg := p.Config.GetTemporalConfig()

	clientOptions, summary, err := connection.BuildOptions(cfg)
	if err != nil {
		return TemporalClientResult{}, fmt.Errorf("configure temporal client: %w", err)
	}
	connectionFields := summary.Fields()

	if cfg.Security.EnableEncryption || cfg.Security.EnableCompression {
		log.Info("configuring Temporal data converter with security features",
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
				"encryption enabled - ensure TEMPORAL_ENCRYPTION_KEY environment variable is set",
				zap.String("keyID", cfg.Security.EncryptionKeyID),
			)
		}
	}

	c, err := client.Dial(clientOptions)
	if err != nil {
		log.Warn(
			"failed to connect to temporal - workflows will be disabled",
			append(connectionFields, zap.Error(err))...,
		)
		return TemporalClientResult{Client: nil}, nil
	}

	log.Info(
		"temporal client connected",
		append(
			connectionFields,
			zap.String("identity", clientOptions.Identity),
			zap.Bool("encryptionEnabled", cfg.Security.EnableEncryption),
			zap.Bool("compressionEnabled", cfg.Security.EnableCompression),
		)...,
	)

	p.LC.Append(fx.Hook{
		OnStop: func(context.Context) error {
			log.Info("closing temporal client")
			c.Close()
			return nil
		},
	})

	return TemporalClientResult{Client: c}, nil
}
