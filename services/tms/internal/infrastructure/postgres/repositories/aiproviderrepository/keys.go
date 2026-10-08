package aiproviderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type KeyParams struct {
	fx.In

	DB *postgres.Connection
}

type keyRepository struct {
	db *postgres.Connection
}

func NewKeyRepository(p KeyParams) repositories.AIProviderKeyRepository {
	return &keyRepository{db: p.DB}
}

func (r *keyRepository) RecordFingerprint(
	ctx context.Context,
	req repositories.RecordAIProviderKeyFingerprintRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.ProviderColumns

		if _, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*aiprovider.Provider)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ProviderScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.APIKey.IsNotNull()).
					Where(cols.APIKeyPrefix.IsNull()).
					Where(cols.APIKeyLastFour.IsNull())
			}).
			Set(cols.APIKeyPrefix.Set(), stringutils.Ptr(req.Prefix)).
			Set(cols.APIKeyLastFour.Set(), stringutils.Ptr(req.LastFour)).
			Exec(ctx); err != nil {
			return fmt.Errorf("record ai provider key fingerprint: %w", err)
		}

		return nil
	})
}

func (r *keyRepository) TouchUsed(
	ctx context.Context,
	req repositories.TouchAIProviderKeyRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.ProviderColumns

		if _, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*aiprovider.Provider)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ProviderScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.APIKey.IsNotNull()).
					WhereGroup(" AND ", func(inner *bun.UpdateQuery) *bun.UpdateQuery {
						return inner.Where(cols.APIKeyLastUsedAt.IsNull()).
							WhereOr(cols.APIKeyLastUsedAt.Lt(), req.UsedAt)
					})
			}).
			Set(cols.APIKeyLastUsedAt.Set(), req.UsedAt).
			Exec(ctx); err != nil {
			return fmt.Errorf("record ai provider key use: %w", err)
		}

		return nil
	})
}

func (r *keyRepository) ClearExpiredPrevious(
	ctx context.Context,
	req repositories.ClearExpiredAIProviderKeysRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := buncolgen.ProviderColumns

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*aiprovider.Provider)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ProviderScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.RotationExpiresAt.IsNotNull()).
					Where(cols.RotationExpiresAt.Lte(), req.Now)
			}).
			Set(cols.PreviousAPIKey.SetNull()).
			Set(cols.RotationExpiresAt.SetNull()).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("clear expired ai provider keys: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("count cleared ai provider keys: %w", err)
		}

		return int(affected), nil
	})
}
