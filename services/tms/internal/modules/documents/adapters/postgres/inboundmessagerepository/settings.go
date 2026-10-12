//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type targetRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewTargetRepository(p Params) inboundmessageports.TriageTargetRepository {
	return &targetRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-target-repository"),
	}
}

func (r *targetRepository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*triage.ResponseTarget, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.ResponseTarget, error) {
		targets := make([]*triage.ResponseTarget, 0, 8)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&targets).
			Apply(triagecol.ResponseTargetApplyTenant(tenantInfo)).
			Order(triagecol.ResponseTargetColumns.Classification.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		return targets, nil
	})
}

func (r *targetRepository) Save(ctx context.Context, req inboundmessageports.SaveTargetsRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		dba := r.db.DBForContext(ctx)
		cols := triagecol.ResponseTargetColumns
		if len(req.Reset) > 0 {
			if _, err := dba.NewDelete().
				Model((*triage.ResponseTarget)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return triagecol.ResponseTargetScopeTenantDelete(dq, req.TenantInfo).
						Where(cols.Classification.In(), bun.List(req.Reset))
				}).
				Exec(ctx); err != nil {
				return err
			}
		}
		if len(req.Upsert) == 0 {
			return nil
		}

		_, err := dba.NewInsert().
			Model(&req.Upsert).
			On("CONFLICT (organization_id, business_unit_id, classification) DO UPDATE").
			Set(cols.Minutes.SetExcluded()).
			Set(cols.UpdatedByID.SetExcluded()).
			Set(cols.UpdatedAt.SetExcluded()).
			Exec(ctx)

		return err
	})
}

type settingsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSettingsRepository(p Params) inboundmessageports.TriageSettingsRepository {
	return &settingsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-settings-repository"),
	}
}

func (r *settingsRepository) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*triage.Settings, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Settings, error) {
		settings := new(triage.Settings)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(settings).
			Apply(triagecol.SettingsApplyTenant(tenantInfo)).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return triage.DefaultSettings(tenantInfo.OrgID, tenantInfo.BuID), nil
		}
		if err != nil {
			return nil, err
		}

		return settings, nil
	})
}

func (r *settingsRepository) Save(
	ctx context.Context,
	settings *triage.Settings,
) (*triage.Settings, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Settings, error) {
		dba := r.db.DBForContext(ctx)
		ov := settings.Version
		settings.Version++
		cols := triagecol.SettingsColumns

		if ov == 0 {
			results, err := dba.NewInsert().
				Model(settings).
				On("CONFLICT (organization_id, business_unit_id) DO NOTHING").
				Returning("*").
				Exec(ctx)
			if err != nil {
				settings.Version = ov
				return nil, err
			}
			if affected, aErr := results.RowsAffected(); aErr == nil && affected > 0 {
				return settings, nil
			}
		}

		results, err := dba.NewUpdate().
			Model(settings).
			WherePK().
			Where(cols.Version.Eq(), ov).
			ExcludeColumn(cols.CreatedAt.Bare()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			settings.Version = ov
			return nil, err
		}
		affected, err := results.RowsAffected()
		if err != nil {
			settings.Version = ov
			return nil, err
		}
		if affected == 0 {
			settings.Version = ov
			return nil, errortypes.NewConflictError(
				"Someone else changed these settings. Reload them and try again.",
			)
		}

		return settings, nil
	})
}
