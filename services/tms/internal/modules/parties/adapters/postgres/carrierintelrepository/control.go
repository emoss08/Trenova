package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
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
) (*carrierintelcontrol.CarrierIntelControl, error) {
	c := carrierinteldb.CarrierIntelControls
	newDefault := func() *carrierintelcontrol.CarrierIntelControl {
		return carrierintelcontrol.NewDefaultControl(tenantInfo.OrgID, tenantInfo.BuID)
	}

	return dbtx.GetOrCreate(
		ctx,
		r.db,
		dbtx.GetOrCreateSpec[*carrierintelcontrol.CarrierIntelControl]{
			Find: func(ctx context.Context) (*carrierintelcontrol.CarrierIntelControl, error) {
				return c.Cache(resultcache.Settings()...).One(ctx, r.db.Keel(ctx))
			},
			Create: func(ctx context.Context) error {
				if _, err := c.CreateOrSkip(
					ctx,
					r.db.Keel(ctx),
					[]*carrierintelcontrol.CarrierIntelControl{newDefault()},
					c.OrganizationID,
					c.BusinessUnitID,
				); err != nil {
					r.l.Error("failed to create default carrier intel control", zap.Error(err))
					return fmt.Errorf("create default carrier intel control: %w", err)
				}

				return nil
			},
			Default: newDefault,
			MapError: func(err error) error {
				r.l.Error("failed to get carrier intel control", zap.Error(err))
				return controlErrors.Map(err, tenantInfo.OrgID.String())
			},
		},
	)
}

func (r *controlRepository) Update(
	ctx context.Context,
	entity *carrierintelcontrol.CarrierIntelControl,
) (*carrierintelcontrol.CarrierIntelControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelcontrol.CarrierIntelControl, error) {
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
) ([]*carrierintelcontrol.CarrierIntelControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with carrier intelligence configured")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintelcontrol.CarrierIntelControl, error) {
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
				q = q.Where(
					keel.Tuple(c.OrganizationID, c.BusinessUnitID).
						Gt(keel.Tuple(req.After.OrganizationID.String(), req.After.BusinessUnitID.String())),
				)
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
