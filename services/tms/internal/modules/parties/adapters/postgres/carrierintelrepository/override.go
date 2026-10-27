package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	carrierOverrideHistoryLimit = 100
	overrideEntityName          = "CarrierIntelOverride"
)

var overrideErrors = dberror.Mapping{Entity: overrideEntityName}

type overrideRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewOverrideRepository(p Params) carrierintelports.CarrierIntelOverrideRepository {
	return &overrideRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-override-repository"),
	}
}

func (r *overrideRepository) ListActiveByCarrierIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	carrierIDs []pulid.ID,
	now int64,
) ([]*carrierintel.CarrierIntelOverride, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierIntelOverride, error) {
			if len(carrierIDs) == 0 {
				return []*carrierintel.CarrierIntelOverride{}, nil
			}

			o := carrierinteldb.CarrierIntelOverrides
			entities, err := o.Where(
				o.CarrierID.In(carrierIDs...),
				o.RevokedAt.IsNull(),
				o.ExpiresAt.Gt(now),
			).
				Order(o.GrantedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list active carrier intel overrides", zap.Error(err))
				return nil, fmt.Errorf("list active carrier intel overrides: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *overrideRepository) ListByCarrier(
	ctx context.Context,
	_ pagination.TenantInfo,
	carrierID pulid.ID,
) ([]*carrierintel.CarrierIntelOverride, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierIntelOverride, error) {
			o := carrierinteldb.CarrierIntelOverrides
			entities, err := o.Where(o.CarrierID.Eq(carrierID)).
				Order(o.GrantedAt.Desc(), o.ID.Desc()).
				Limit(carrierOverrideHistoryLimit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list carrier intel overrides", zap.Error(err))
				return nil, fmt.Errorf("list carrier intel overrides: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *overrideRepository) GetByID(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*carrierintel.CarrierIntelOverride, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierIntelOverride, error) {
			entity, err := carrierinteldb.CarrierIntelOverrides.Get(ctx, r.db.Keel(ctx), id)
			if err != nil {
				return nil, overrideErrors.Map(err, id.String())
			}

			return entity, nil
		},
	)
}

func (r *overrideRepository) Create(
	ctx context.Context,
	entity *carrierintel.CarrierIntelOverride,
) (*carrierintel.CarrierIntelOverride, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierIntelOverride, error) {
			err := carrierinteldb.CarrierIntelOverrides.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create carrier intel override", zap.Error(err))
				return nil, fmt.Errorf("create carrier intel override: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *overrideRepository) Update(
	ctx context.Context,
	entity *carrierintel.CarrierIntelOverride,
) (*carrierintel.CarrierIntelOverride, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierIntelOverride, error) {
			err := carrierinteldb.CarrierIntelOverrides.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update carrier intel override", zap.Error(err))
				return nil, saveError(
					overrideErrors,
					err,
					entity.ID.String(),
					"update carrier intel override",
				)
			}

			return entity, nil
		},
	)
}
