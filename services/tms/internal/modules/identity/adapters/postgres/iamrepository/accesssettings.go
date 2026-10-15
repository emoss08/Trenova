package iamrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/buncolgen/iamcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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
		cols := iamcol.AccessSettingsColumns
		settings := new(iam.AccessSettings)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(settings).
			Where(cols.OrganizationID.Eq(), tenant.OrgID).
			Where(cols.BusinessUnitID.Eq(), tenant.BuID).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return iam.DefaultAccessSettings(tenant.OrgID, tenant.BuID), nil
			}
			r.l.Error("failed to read access settings", zap.Error(err))
			return nil, fmt.Errorf("read access settings: %w", err)
		}

		return settings, nil
	})
}

func (r *accessSettingsRepository) Save(
	ctx context.Context,
	settings *iam.AccessSettings,
) (*iam.AccessSettings, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.AccessSettings, error) {
		cols := iamcol.AccessSettingsColumns
		expected := settings.Version
		settings.Version = expected + 1

		res, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(settings).
			On("CONFLICT (organization_id, business_unit_id) DO UPDATE").
			Set(cols.RequireRoleActivation.Set(), settings.RequireRoleActivation).
			Set(cols.UpdatedByID.Set(), settings.UpdatedByID).
			Set(cols.Version.Set(), settings.Version).
			Set(cols.UpdatedAt.Set(), settings.UpdatedAt).
			Where(cols.Version.Qualified()+" = ?", expected).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to save access settings", zap.Error(err))
			return nil, fmt.Errorf("save access settings: %w", err)
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return nil, errortypes.NewValidationError(
				"version",
				errortypes.ErrVersionMismatch,
				"These settings were changed by someone else. Reload them and try again.",
			)
		}

		return settings, nil
	})
}
