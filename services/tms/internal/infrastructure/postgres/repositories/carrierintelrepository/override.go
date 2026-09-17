package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	carrierOverrideHistoryLimit = 100
	overrideEntityName          = "CarrierIntelOverride"
)

type overrideRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewOverrideRepository(p Params) repositories.CarrierIntelOverrideRepository {
	return &overrideRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-override-repository"),
	}
}

func (r *overrideRepository) ListActiveByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
	now int64,
) ([]*carrierintel.CarrierIntelOverride, error) {
	if len(carrierIDs) == 0 {
		return []*carrierintel.CarrierIntelOverride{}, nil
	}

	cols := buncolgen.CarrierIntelOverrideColumns
	entities := make([]*carrierintel.CarrierIntelOverride, 0, len(carrierIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelOverrideScopeTenant(sq, tenantInfo).
				Where(cols.CarrierID.In(), bun.List(carrierIDs)).
				Where(cols.RevokedAt.IsNull()).
				Where(cols.ExpiresAt.Gt(), now)
		}).
		Order(cols.GrantedAt.OrderDesc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list active carrier intel overrides", zap.Error(err))
		return nil, fmt.Errorf("list active carrier intel overrides: %w", err)
	}

	return entities, nil
}

func (r *overrideRepository) ListByCarrier(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) ([]*carrierintel.CarrierIntelOverride, error) {
	cols := buncolgen.CarrierIntelOverrideColumns
	entities := make([]*carrierintel.CarrierIntelOverride, 0, carrierOverrideHistoryLimit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelOverrideScopeTenant(sq, tenantInfo).
				Where(cols.CarrierID.Eq(), carrierID)
		}).
		Order(cols.GrantedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(carrierOverrideHistoryLimit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier intel overrides", zap.Error(err))
		return nil, fmt.Errorf("list carrier intel overrides: %w", err)
	}

	return entities, nil
}

func (r *overrideRepository) GetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*carrierintel.CarrierIntelOverride, error) {
	entity := new(carrierintel.CarrierIntelOverride)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelOverrideScopeTenant(sq, tenantInfo).
				Where(buncolgen.CarrierIntelOverrideColumns.ID.Eq(), id)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, overrideEntityName)
	}

	return entity, nil
}

func (r *overrideRepository) Create(
	ctx context.Context,
	entity *carrierintel.CarrierIntelOverride,
) (*carrierintel.CarrierIntelOverride, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create carrier intel override", zap.Error(err))
		return nil, fmt.Errorf("create carrier intel override: %w", err)
	}

	return entity, nil
}

func (r *overrideRepository) Update(
	ctx context.Context,
	entity *carrierintel.CarrierIntelOverride,
) (*carrierintel.CarrierIntelOverride, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.CarrierIntelOverrideColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update carrier intel override", zap.Error(err))
		return nil, fmt.Errorf("update carrier intel override: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, overrideEntityName, entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
