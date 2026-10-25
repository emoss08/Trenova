package iamrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/zap"
)

type accessSettingsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAccessSettingsRepository(p Params) iamports.AccessSettingsRepository {
	return &accessSettingsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.access-settings-repository"),
	}
}

func (r *accessSettingsRepository) Get(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*iam.AccessSettings, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.AccessSettings, error) {
		rows, err := iamdb.AccessSettingses.Limit(1).
			Cache(resultcache.Settings()...).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to read access settings", zap.Error(err))
			return nil, fmt.Errorf("read access settings: %w", err)
		}
		if len(rows) == 0 {
			return iam.DefaultAccessSettings(tenant.OrgID, tenant.BuID), nil
		}

		return rows[0], nil
	})
}

func (r *accessSettingsRepository) Save(
	ctx context.Context,
	settings *iam.AccessSettings,
) (*iam.AccessSettings, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.AccessSettings, error) {
		saved, err := r.write(ctx, settings)
		if err != nil {
			r.l.Error("failed to save access settings", zap.Error(err))
			return nil, fmt.Errorf("save access settings: %w", err)
		}
		if !saved {
			return nil, errortypes.NewValidationError(
				"version",
				errortypes.ErrVersionMismatch,
				"These settings were changed by someone else. Reload them and try again.",
			)
		}

		return settings, nil
	})
}

func (r *accessSettingsRepository) write(
	ctx context.Context,
	settings *iam.AccessSettings,
) (bool, error) {
	as := iamdb.AccessSettingses
	db := r.db.Keel(ctx)
	if settings.Version > 0 {
		err := as.Save(ctx, db, settings, as.RequireRoleActivation, as.UpdatedByID)
		if dberror.IsVersionConflict(err) {
			return false, nil
		}
		return err == nil, err
	}

	settings.Version = 1
	inserted, err := as.CreateOrSkip(
		ctx,
		db,
		[]*iam.AccessSettings{settings},
		as.OrganizationID,
		as.BusinessUnitID,
	)
	if err != nil || inserted == 0 {
		settings.Version = 0
	}
	return inserted == 1, err
}
