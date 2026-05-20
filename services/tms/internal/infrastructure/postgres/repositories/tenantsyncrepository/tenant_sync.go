package tenantsyncrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) repositories.TenantSyncRepository {
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
	return r.listOrganizations(ctx, nil)
}

func (r *repository) ListBusinessUnitsByID(
	ctx context.Context,
	ids []pulid.ID,
) ([]tenant.SyncBusinessUnit, error) {
	if len(ids) == 0 {
		return []tenant.SyncBusinessUnit{}, nil
	}
	return r.listBusinessUnits(ctx, ids)
}

func (r *repository) ListOrganizationsByID(
	ctx context.Context,
	ids []pulid.ID,
) ([]tenant.SyncOrganization, error) {
	if len(ids) == 0 {
		return []tenant.SyncOrganization{}, nil
	}
	return r.listOrganizations(ctx, ids)
}

func (r *repository) listBusinessUnits(
	ctx context.Context,
	ids []pulid.ID,
) ([]tenant.SyncBusinessUnit, error) {
	rows := make([]tenant.SyncBusinessUnit, 0)
	query := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("business_units AS bu").
		ColumnExpr("bu.id").
		ColumnExpr("bu.name").
		ColumnExpr("bu.code").
		ColumnExpr("bu.created_at").
		ColumnExpr("bu.updated_at").
		OrderExpr("bu.id ASC")

	if len(ids) > 0 {
		query = query.Where("bu.id IN (?)", bun.In(ids))
	}

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, err
	}

	for i := range rows {
		rows[i].Metadata = map[string]string{
			"tmsCode": rows[i].Code,
		}
	}

	return rows, nil
}

func (r *repository) listOrganizations(
	ctx context.Context,
	ids []pulid.ID,
) ([]tenant.SyncOrganization, error) {
	rows := make([]tenant.SyncOrganization, 0)
	query := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("organizations AS org").
		ColumnExpr("org.id").
		ColumnExpr("org.business_unit_id").
		ColumnExpr("org.name").
		ColumnExpr("org.login_slug").
		ColumnExpr("org.scac_code").
		ColumnExpr("org.dot_number").
		ColumnExpr("org.created_at").
		ColumnExpr("org.updated_at").
		OrderExpr("org.id ASC")

	if len(ids) > 0 {
		query = query.Where("org.id IN (?)", bun.In(ids))
	}

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, err
	}

	for i := range rows {
		rows[i].Metadata = map[string]string{
			"tmsLoginSlug": rows[i].LoginSlug,
			"tmsScacCode":  rows[i].ScacCode,
			"tmsDotNumber": rows[i].DOTNumber,
		}
	}

	return rows, nil
}
