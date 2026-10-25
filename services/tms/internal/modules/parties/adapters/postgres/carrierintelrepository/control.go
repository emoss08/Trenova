package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"go.uber.org/zap"
)

const (
	defaultConfiguredTenantLimit = 200
	maxConfiguredTenantLimit     = 1000
	controlEntityName            = "CarrierIntelControl"
)

var controlErrors = dberror.Mapping{Entity: controlEntityName}

type controlRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewControlRepository(p Params) carrierintelports.CarrierIntelControlRepository {
	return &controlRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-control-repository"),
	}
}

func (r *controlRepository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carrierintel.CarrierIntelControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierIntelControl, error) {
			c := carrierinteldb.CarrierIntelControls
			db := r.db.Keel(ctx)
			entity, err := c.Query().One(ctx, db)
			if err == nil {
				return entity, nil
			}
			if !dberror.IsNotFoundError(err) {
				r.l.Error("failed to get carrier intel control", zap.Error(err))
				return nil, fmt.Errorf("get carrier intel control: %w", err)
			}

			if _, err = c.CreateOrSkip(
				ctx,
				db,
				[]*carrierintel.CarrierIntelControl{
					carrierintel.NewDefaultControl(tenantInfo.OrgID, tenantInfo.BuID),
				},
				c.OrganizationID,
				c.BusinessUnitID,
			); err != nil {
				r.l.Error("failed to create default carrier intel control", zap.Error(err))
				return nil, fmt.Errorf("create default carrier intel control: %w", err)
			}

			entity, err = c.Query().One(ctx, db)
			if err != nil {
				return nil, controlErrors.Map(err, tenantInfo.OrgID.String())
			}

			return entity, nil
		},
	)
}

func (r *controlRepository) Update(
	ctx context.Context,
	entity *carrierintel.CarrierIntelControl,
) (*carrierintel.CarrierIntelControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierIntelControl, error) {
			err := carrierinteldb.CarrierIntelControls.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update carrier intel control", zap.Error(err))
				return nil, saveError(
					controlErrors,
					err,
					entity.ID.String(),
					"update carrier intel control",
				)
			}

			return entity, nil
		},
	)
}

func (r *controlRepository) ListConfigured(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelTenantsRequest,
) ([]*carrierintel.CarrierIntelControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with carrier intelligence configured")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierIntelControl, error) {
			c := carrierinteldb.CarrierIntelControls
			limit := intutils.Clamp(
				intutils.WithDefault(max(req.Limit, 0), defaultConfiguredTenantLimit),
				1,
				maxConfiguredTenantLimit,
			)

			q := c.AllTenants().
				Where(c.PrimaryProvider.IsNotNull()).
				Order(c.OrganizationID.Asc(), c.BusinessUnitID.Asc()).
				Limit(limit)
			if !req.After.OrganizationID.IsNil() {
				q = q.Where(keel.Raw(
					"(?, ?) > (?, ?)",
					c.OrganizationID,
					c.BusinessUnitID,
					req.After.OrganizationID.String(),
					req.After.BusinessUnitID.String(),
				))
			}

			entities, err := q.List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list configured carrier intel controls", zap.Error(err))
				return nil, fmt.Errorf("list configured carrier intel controls: %w", err)
			}

			return entities, nil
		},
	)
}
