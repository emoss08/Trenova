package aiproviderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"go.uber.org/fx"
)

type KeyParams struct {
	fx.In

	DB *postgres.Connection
}

type keyRepository struct {
	db *postgres.Connection
}

func NewKeyRepository(p KeyParams) aiproviderports.AIProviderKeyRepository {
	return &keyRepository{db: p.DB}
}

func (r *keyRepository) RecordFingerprint(
	ctx context.Context,
	req *aiproviderports.RecordAIProviderKeyFingerprintRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := aiproviderdb.Providers
		if _, err := p.Where(
			p.ID.Eq(req.ID),
			p.APIKey.IsNotNull(),
			p.APIKeyPrefix.IsNull(),
			p.APIKeyLastFour.IsNull(),
		).Update(ctx, r.db.Keel(ctx), append(untouched(),
			p.APIKeyPrefix.To(req.Prefix),
			p.APIKeyLastFour.To(req.LastFour),
		)...); err != nil {
			return fmt.Errorf("record ai provider key fingerprint: %w", err)
		}

		return nil
	})
}

func (r *keyRepository) TouchUsed(
	ctx context.Context,
	req aiproviderports.TouchAIProviderKeyRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := aiproviderdb.Providers
		if _, err := p.Where(
			p.ID.Eq(req.ID),
			p.APIKey.IsNotNull(),
			keel.Or(p.APIKeyLastUsedAt.IsNull(), p.APIKeyLastUsedAt.Lt(req.UsedAt)),
		).Update(ctx, r.db.Keel(ctx), append(untouched(),
			p.APIKeyLastUsedAt.To(req.UsedAt),
		)...); err != nil {
			return fmt.Errorf("record ai provider key use: %w", err)
		}

		return nil
	})
}

func (r *keyRepository) ClearExpiredPrevious(
	ctx context.Context,
	req aiproviderports.ClearExpiredAIProviderKeysRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		p := aiproviderdb.Providers
		affected, err := p.Where(
			p.RotationExpiresAt.IsNotNull(),
			p.RotationExpiresAt.Lte(req.Now),
		).Update(ctx, r.db.Keel(ctx), append(untouched(),
			p.PreviousAPIKey.ToNull(),
			p.RotationExpiresAt.ToNull(),
		)...)
		if err != nil {
			return 0, fmt.Errorf("clear expired ai provider keys: %w", err)
		}

		return int(affected), nil
	})
}

func (r *keyRepository) RebindSealedKey(
	ctx context.Context,
	req *aiproviderports.RebindAIProviderKeyRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := aiproviderdb.Providers
		column := p.APIKey
		if req.Previous {
			column = p.PreviousAPIKey
		}

		if _, err := p.Where(p.ID.Eq(req.ID), column.Eq(req.Expected)).
			Update(ctx, r.db.Keel(ctx), append(untouched(), column.To(req.Replacement))...); err != nil {
			return fmt.Errorf("rebind ai provider key: %w", err)
		}

		return nil
	})
}
