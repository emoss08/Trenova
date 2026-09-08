package orgstructurerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// Two rosters hold positions. Drivers are workers and carry position_id on
// their own row; the front office is made of people who log in, and their
// title sits on their membership in the organisation. Everything here reads
// both so the chart never shows a desk as empty because the person at it has
// a login rather than a CDL.

func (r *repository) StaffByPosition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.StaffCountRow, error) {
	rows := make([]repositories.StaffCountRow, 0, 16)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.OrganizationMembership)(nil)).
		Join("JOIN users AS usr").
		JoinOn("usr.id = uom.user_id").
		Join("JOIN job_positions AS jpos").
		JoinOn("jpos.id = uom.position_id").
		JoinOn("jpos.organization_id = uom.organization_id").
		JoinOn("jpos.business_unit_id = uom.business_unit_id").
		Where("uom.organization_id = ?", tenantInfo.OrgID).
		Where("uom.business_unit_id = ?", tenantInfo.BuID).
		Where("usr.status = 'Active'").
		ColumnExpr("uom.position_id AS position_id").
		ColumnExpr("jpos.title AS title").
		ColumnExpr("jpos.code AS code").
		ColumnExpr("jpos.department::text AS department").
		ColumnExpr("COUNT(*) AS staff").
		GroupExpr("uom.position_id, jpos.title, jpos.code, jpos.department").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to count staff by position", zap.Error(err))
		return nil, fmt.Errorf("count staff by position: %w", err)
	}
	return rows, nil
}

func (r *repository) ListPositionHolders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	positionID pulid.ID,
) ([]repositories.PositionHolderRow, error) {
	workers := make([]repositories.PositionHolderRow, 0, 16)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		Join("LEFT JOIN fleet_codes AS fc").
		JoinOn("fc.id = wrk.fleet_code_id").
		JoinOn("fc.organization_id = wrk.organization_id").
		JoinOn("fc.business_unit_id = wrk.business_unit_id").
		Where("wrk.organization_id = ?", tenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", tenantInfo.BuID).
		Where("wrk.position_id = ?", positionID).
		ColumnExpr("? AS kind", string(repositories.PositionHolderWorker)).
		ColumnExpr("wrk.id AS id").
		ColumnExpr("wrk.first_name || ' ' || wrk.last_name AS name").
		ColumnExpr("wrk.status::text AS status").
		ColumnExpr("COALESCE(fc.code, '') AS detail").
		OrderExpr("wrk.status = 'Active' DESC, wrk.last_name, wrk.first_name").
		Scan(ctx, &workers); err != nil {
		r.l.Error("failed to list workers in position", zap.Error(err))
		return nil, fmt.Errorf("list workers in position: %w", err)
	}

	users := make([]repositories.PositionHolderRow, 0, 8)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.OrganizationMembership)(nil)).
		Join("JOIN users AS usr").
		JoinOn("usr.id = uom.user_id").
		Where("uom.organization_id = ?", tenantInfo.OrgID).
		Where("uom.business_unit_id = ?", tenantInfo.BuID).
		Where("uom.position_id = ?", positionID).
		ColumnExpr("? AS kind", string(repositories.PositionHolderUser)).
		ColumnExpr("usr.id AS id").
		ColumnExpr("usr.name AS name").
		ColumnExpr("usr.status::text AS status").
		ColumnExpr("usr.email_address AS detail").
		OrderExpr("usr.status = 'Active' DESC, usr.name").
		Scan(ctx, &users); err != nil {
		r.l.Error("failed to list users in position", zap.Error(err))
		return nil, fmt.Errorf("list users in position: %w", err)
	}

	return append(workers, users...), nil
}

func (r *repository) SetWorkerPosition(
	ctx context.Context,
	req *repositories.SetWorkerPositionRequest,
) error {
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.Worker)(nil)).
		Set("position_id = ?", nullableID(req.PositionID)).
		Set("updated_at = ?", timeutils.NowUnix()).
		Where("id = ?", req.WorkerID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to set worker position", zap.Error(err))
		return fmt.Errorf("set worker position: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return errortypes.NewNotFoundError("Worker not found")
	}
	return nil
}

func (r *repository) SetUserPosition(
	ctx context.Context,
	req *repositories.SetUserPositionRequest,
) error {
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tenant.OrganizationMembership)(nil)).
		Set("position_id = ?", nullableID(req.PositionID)).
		Where("user_id = ?", req.UserID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to set user position", zap.Error(err))
		return fmt.Errorf("set user position: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return errortypes.NewNotFoundError("That user is not a member of this organisation")
	}
	return nil
}

// nullableID writes an empty id as NULL rather than as an empty string, which
// a foreign key would refuse.
func nullableID(id pulid.ID) any {
	if id.IsNil() {
		return nil
	}
	return id
}
