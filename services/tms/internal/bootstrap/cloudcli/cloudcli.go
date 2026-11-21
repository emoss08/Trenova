package cloudcli

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/bootstrap/envelopes"
	"github.com/emoss08/trenova/internal/cloud/cli/archiveimport"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/instancebootstraprepository"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const finalizeReason = "record a dedicated instance as set up from the tenant archive it was made from, before any tenant scope exists"

func ImportArchive(cfg func() *config.Config) *cobra.Command {
	return archiveimport.Command(func(ctx context.Context) (*archiveimport.Deps, error) {
		c := cfg()
		logger, _, err := config.ProvideLogger(&c.Logging)
		if err != nil {
			return nil, err
		}
		encryption, err := encryptionservice.New(
			encryptionservice.Params{SecurityConfig: &c.Security},
		)
		if err != nil {
			return nil, fmt.Errorf("load the encryption keys: %w", err)
		}
		db, err := postgres.OpenMaintenanceDB(ctx, &c.App, &c.Database)
		if err != nil {
			return nil, fmt.Errorf("open the database: %w", err)
		}
		return &archiveimport.Deps{
			Config:    c,
			DB:        db,
			Secrets:   encryptionservice.NewSecretCipher(encryption),
			Blobs:     encryptionservice.NewBlobCipher(encryption),
			Envelopes: envelopes.Sources(true),
			Finalize:  finalize(logger),
			Logger:    logger,
			Close:     db.Close,
		}, nil
	})
}

func finalize(logger *zap.Logger) archiveimport.Finalizer {
	return func(
		ctx context.Context,
		conn *postgres.Connection,
		req *instancebootstrapports.FinalizeInstanceBootstrapRequest,
	) error {
		repo := instancebootstraprepository.New(instancebootstraprepository.Params{
			DB: conn, Logger: logger,
		})
		_, err := repo.Finalize(dbscope.WithSystem(ctx, finalizeReason), req)
		return err
	}
}
