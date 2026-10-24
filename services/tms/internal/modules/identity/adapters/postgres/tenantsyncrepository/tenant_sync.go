package tenantsyncrepository

import (
	"context"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) tenantports.TenantSyncRepository {
	return &repository{db: p.DB}
}

func (r *repository) ListBusinessUnits(
	ctx context.Context,
) ([]tenant.SyncBusinessUnit, error) {
	return r.listBusinessUnits(ctx, nil)
}

func (r *repository) ListOrganizations(
	ctx context.Context,
) ([]tenant.SyncOrganization, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list organizations across every tenant for fan-out and control plane sync",
	)
	return r.listOrganizations(ctx, nil)
}

func (r *repository) ListBusinessUnitsByID(
	ctx context.Context,
	ids []pulid.ID,
) ([]tenant.SyncBusinessUnit, error) {
	ctx = dbscope.WithSystem(ctx, "load business units for control plane sync")
	if len(ids) == 0 {
		return []tenant.SyncBusinessUnit{}, nil
	}
	return r.listBusinessUnits(ctx, ids)
}

func (r *repository) ListOrganizationsByID(
	ctx context.Context,
	ids []pulid.ID,
) ([]tenant.SyncOrganization, error) {
	ctx = dbscope.WithSystem(ctx, "load organizations across tenants for control plane sync")
	if len(ids) == 0 {
		return []tenant.SyncOrganization{}, nil
	}
	return r.listOrganizations(ctx, ids)
}

func (r *repository) listBusinessUnits(
	ctx context.Context,
	ids []pulid.ID,
) ([]tenant.SyncBusinessUnit, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]tenant.SyncBusinessUnit, error) {
		bu := tenantdb.BusinessUnits
		units, err := bu.Pick(bu.ID, bu.Name, bu.Code, bu.CreatedAt, bu.UpdatedAt).
			Where(keel.When(len(ids) > 0, bu.ID.In(ids...))).
			Order(bu.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		rows := make([]tenant.SyncBusinessUnit, 0, len(units))
		for _, unit := range units {
			rows = append(rows, tenant.SyncBusinessUnit{
				ID:        unit.ID,
				Name:      unit.Name,
				Code:      unit.Code,
				Metadata:  map[string]string{"tmsCode": unit.Code},
				CreatedAt: unit.CreatedAt,
				UpdatedAt: unit.UpdatedAt,
			})
		}

		return rows, nil
	})
}

func (r *repository) listOrganizations(
	ctx context.Context,
	ids []pulid.ID,
) ([]tenant.SyncOrganization, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]tenant.SyncOrganization, error) {
		o := tenantdb.Organizations
		orgs, err := o.AllTenants().
			Pick(
				o.ID,
				o.BusinessUnitID,
				o.Name,
				o.LoginSlug,
				o.ScacCode,
				o.DOTNumber,
				o.CreatedAt,
				o.UpdatedAt,
			).
			Where(keel.When(len(ids) > 0, o.ID.In(ids...))).
			Order(o.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		rows := make([]tenant.SyncOrganization, 0, len(orgs))
		for _, org := range orgs {
			rows = append(rows, tenant.SyncOrganization{
				ID:             org.ID,
				BusinessUnitID: org.BusinessUnitID,
				Name:           org.Name,
				LoginSlug:      org.LoginSlug,
				ScacCode:       org.ScacCode,
				DOTNumber:      org.DOTNumber,
				Metadata: map[string]string{
					"tmsLoginSlug": org.LoginSlug,
					"tmsScacCode":  org.ScacCode,
					"tmsDotNumber": org.DOTNumber,
				},
				CreatedAt: org.CreatedAt,
				UpdatedAt: org.UpdatedAt,
			})
		}

		return rows, nil
	})
}
