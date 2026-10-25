//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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
	_ pagination.TenantInfo,
) ([]*triage.ResponseTarget, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.ResponseTarget, error) {
		t := triagedb.ResponseTargets
		targets, err := t.Query().Order(t.Classification.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if targets == nil {
			targets = []*triage.ResponseTarget{}
		}

		return targets, nil
	})
}

func (r *targetRepository) Save(
	ctx context.Context,
	req inboundmessageports.SaveTargetsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		t := triagedb.ResponseTargets
		if len(req.Reset) > 0 {
			if _, err := t.Where(t.Classification.In(req.Reset...)).Delete(ctx, db); err != nil {
				return err
			}
		}
		if len(req.Upsert) == 0 {
			return nil
		}

		_, err := t.Upsert(
			ctx,
			db,
			req.Upsert,
			keel.OnConflict(t.OrganizationID, t.BusinessUnitID, t.Classification).
				Update(t.Minutes, t.UpdatedByID),
		)

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
		settings, err := triagedb.Settingses.Query().One(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
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
		db := r.db.Keel(ctx)
		s := triagedb.Settingses
		ov := settings.Version

		if ov == 0 {
			settings.Version = 1
			inserted, err := s.CreateOrSkip(
				ctx,
				db,
				[]*triage.Settings{settings},
				s.OrganizationID,
				s.BusinessUnitID,
			)
			if err != nil {
				settings.Version = ov
				return nil, err
			}
			if inserted > 0 {
				return settings, nil
			}
			settings.Version = ov
		}

		if err := s.Save(ctx, db, settings); err != nil {
			settings.Version = ov
			if dberror.IsVersionConflict(err) {
				return nil, errortypes.NewConflictError(
					"Someone else changed these settings. Reload them and try again.",
				)
			}

			return nil, err
		}

		return settings, nil
	})
}
