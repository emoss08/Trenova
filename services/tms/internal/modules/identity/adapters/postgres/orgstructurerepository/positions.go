package orgstructurerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	textSQL        = "?::text"
	isActiveSQL    = "? = 'Active'"
	orEmptySQL     = "COALESCE(?, '')"
	workerNameSQL  = "? || ' ' || ?"
	positionsAlias = "jpos"
)

// Two rosters hold positions. Drivers are workers and carry position_id on
// their own row; the front office is made of people who log in, and their
// title sits on their membership in the organisation. Everything here reads
// both so the chart never shows a desk as empty because the person at it has
// a login rather than a CDL.

func (r *repository) StaffByPosition(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]workerports.StaffCountRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.StaffCountRow, error) {
		m := tenantdb.OrganizationMemberships
		u := m.User()
		jp := workerdb.JobPositions
		rows, err := keel.Into[workerports.StaffCountRow](ctx, r.db.Keel(ctx), m.
			Join(jp, jp.ID.EqCol(m.PositionID)).
			Where(u.Status.Eq(domaintypes.StatusActive)).
			Group(m.PositionID, jp.Title, jp.Code, jp.Department).
			Pick(m.PositionID, jp.Title, jp.Code, keel.Raw(textSQL, jp.Department), keel.Count()))
		if err != nil {
			r.l.Error("failed to count staff by position", zap.Error(err))
			return nil, fmt.Errorf("count staff by position: %w", err)
		}
		return rows, nil
	})
}

func (r *repository) ListPositionHolders(
	ctx context.Context,
	_ pagination.TenantInfo,
	positionID pulid.ID,
) ([]workerports.PositionHolderRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.PositionHolderRow, error) {
		db := r.db.Keel(ctx)
		w := workerdb.Workers
		fc := w.FleetCode()
		workers, err := keel.Into[workerports.PositionHolderRow](ctx, db, w.
			Where(w.PositionID.Eq(positionID)).
			Pick(
				keel.Raw(textSQL, string(workerports.PositionHolderWorker)),
				w.ID,
				keel.Raw(workerNameSQL, w.FirstName, w.LastName),
				keel.Raw(textSQL, w.Status),
				keel.Raw(orEmptySQL, fc.Code),
			).
			Order(keel.Desc(keel.Raw(isActiveSQL, w.Status)), w.LastName.Asc(), w.FirstName.Asc()))
		if err != nil {
			r.l.Error("failed to list workers in position", zap.Error(err))
			return nil, fmt.Errorf("list workers in position: %w", err)
		}

		m := tenantdb.OrganizationMemberships
		u := m.User()
		users, err := keel.Into[workerports.PositionHolderRow](ctx, db, m.
			Where(m.PositionID.Eq(positionID), u.ID.IsNotNull()).
			Pick(
				keel.Raw(textSQL, string(workerports.PositionHolderUser)),
				u.ID,
				u.Name,
				keel.Raw(textSQL, u.Status),
				u.EmailAddress,
			).
			Order(keel.Desc(keel.Raw(isActiveSQL, u.Status)), u.Name.Asc()))
		if err != nil {
			r.l.Error("failed to list users in position", zap.Error(err))
			return nil, fmt.Errorf("list users in position: %w", err)
		}

		return append(workers, users...), nil
	})
}

func (r *repository) SetWorkerPosition(
	ctx context.Context,
	req *workerports.SetWorkerPositionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		w := workerdb.Workers
		affected, err := w.Where(w.ID.Eq(req.WorkerID)).
			Update(ctx, r.db.Keel(ctx), w.PositionID.To(req.PositionID))
		if err != nil {
			r.l.Error("failed to set worker position", zap.Error(err))
			return fmt.Errorf("set worker position: %w", err)
		}
		if affected == 0 {
			return errortypes.NewNotFoundError("Worker not found")
		}
		return nil
	})
}

func (r *repository) SetUserPosition(
	ctx context.Context,
	req *workerports.SetUserPositionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		m := tenantdb.OrganizationMemberships
		affected, err := m.Where(m.UserID.Eq(req.UserID)).
			Update(ctx, r.db.Keel(ctx), m.PositionID.To(req.PositionID))
		if err != nil {
			r.l.Error("failed to set user position", zap.Error(err))
			return fmt.Errorf("set user position: %w", err)
		}
		if affected == 0 {
			return errortypes.NewNotFoundError("That user is not a member of this organisation")
		}
		return nil
	})
}
