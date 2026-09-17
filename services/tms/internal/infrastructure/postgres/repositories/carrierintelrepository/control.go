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
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultConfiguredTenantLimit = 200
	maxConfiguredTenantLimit     = 1000
	controlEntityName            = "CarrierIntelControl"
)

type controlRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewControlRepository(p Params) repositories.CarrierIntelControlRepository {
	return &controlRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-control-repository"),
	}
}

func buildDefaultControlInsert(db bun.IDB, tenantInfo pagination.TenantInfo) *bun.InsertQuery {
	return db.NewInsert().
		Model(carrierintel.NewDefaultControl(tenantInfo.OrgID, tenantInfo.BuID)).
		On("CONFLICT (organization_id, business_unit_id) DO NOTHING")
}

func (r *controlRepository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carrierintel.CarrierIntelControl, error) {
	entity, err := r.selectControl(ctx, tenantInfo)
	if err == nil {
		return entity, nil
	}
	if !dberror.IsNotFoundError(err) {
		r.l.Error("failed to get carrier intel control", zap.Error(err))
		return nil, fmt.Errorf("get carrier intel control: %w", err)
	}

	if _, err = buildDefaultControlInsert(r.db.DBForContext(ctx), tenantInfo).Exec(ctx); err != nil {
		r.l.Error("failed to create default carrier intel control", zap.Error(err))
		return nil, fmt.Errorf("create default carrier intel control: %w", err)
	}

	entity, err = r.selectControl(ctx, tenantInfo)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, controlEntityName)
	}

	return entity, nil
}

func (r *controlRepository) selectControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carrierintel.CarrierIntelControl, error) {
	entity := new(carrierintel.CarrierIntelControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelControlScopeTenant(sq, tenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *controlRepository) Update(
	ctx context.Context,
	entity *carrierintel.CarrierIntelControl,
) (*carrierintel.CarrierIntelControl, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.CarrierIntelControlColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update carrier intel control", zap.Error(err))
		return nil, fmt.Errorf("update carrier intel control: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, controlEntityName, entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *controlRepository) ListConfigured(
	ctx context.Context,
	req *repositories.ListCarrierIntelTenantsRequest,
) ([]*carrierintel.CarrierIntelControl, error) {
	cols := buncolgen.CarrierIntelControlColumns
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultConfiguredTenantLimit),
		1,
		maxConfiguredTenantLimit,
	)

	entities := make([]*carrierintel.CarrierIntelControl, 0, limit)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.PrimaryProvider.IsNotNull()).
		Order(cols.OrganizationID.OrderAsc(), cols.BusinessUnitID.OrderAsc()).
		Limit(limit)
	if !req.After.OrganizationID.IsNil() {
		q = q.Where(
			buncolgen.Expr("({0}, {1}) > (?, ?)", cols.OrganizationID, cols.BusinessUnitID),
			req.After.OrganizationID,
			req.After.BusinessUnitID,
		)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list configured carrier intel controls", zap.Error(err))
		return nil, fmt.Errorf("list configured carrier intel controls: %w", err)
	}

	return entities, nil
}
