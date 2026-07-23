package driverpayrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type workerPayAssignmentRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewWorkerPayAssignment(p Params) repositories.WorkerPayAssignmentRepository {
	return &workerPayAssignmentRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-pay-assignment-repository"),
	}
}

func (r *workerPayAssignmentRepository) GetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*driverpay.WorkerPayAssignment, error) {
	entity := new(driverpay.WorkerPayAssignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("wpa.id = ?", id).
		Where("wpa.organization_id = ?", tenantInfo.OrgID).
		Where("wpa.business_unit_id = ?", tenantInfo.BuID).
		Relation("PayProfile", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerPayAssignment")
	}
	return entity, nil
}

func (r *workerPayAssignmentRepository) GetEffectiveForWorker(
	ctx context.Context,
	req repositories.GetWorkerPayAssignmentRequest,
) (*driverpay.WorkerPayAssignment, error) {
	entity := new(driverpay.WorkerPayAssignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("wpa.organization_id = ?", req.TenantInfo.OrgID).
		Where("wpa.business_unit_id = ?", req.TenantInfo.BuID).
		Where("wpa.worker_id = ?", req.WorkerID).
		Where("wpa.effective_from <= ?", req.AsOf).
		Where("wpa.effective_to IS NULL OR wpa.effective_to > ?", req.AsOf).
		Relation("PayProfile", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Relation("PayProfile.Components", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("dppc.sequence ASC")
		}).
		Order("wpa.effective_from DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberror.HandleNotFoundError(err, "WorkerPayAssignment")
		}
		return nil, fmt.Errorf("get effective pay assignment: %w", err)
	}
	return entity, nil
}

func (r *workerPayAssignmentRepository) ListForWorker(
	ctx context.Context,
	req repositories.ListWorkerPayAssignmentsRequest,
) ([]*driverpay.WorkerPayAssignment, error) {
	items := make([]*driverpay.WorkerPayAssignment, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("wpa.organization_id = ?", req.TenantInfo.OrgID).
		Where("wpa.business_unit_id = ?", req.TenantInfo.BuID).
		Where("wpa.worker_id = ?", req.WorkerID).
		Relation("PayProfile", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("wpa.effective_from DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list worker pay assignments: %w", err)
	}
	return items, nil
}

func (r *workerPayAssignmentRepository) ListForProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profileID pulid.ID,
) ([]*driverpay.WorkerPayAssignment, error) {
	items := make([]*driverpay.WorkerPayAssignment, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("wpa.organization_id = ?", tenantInfo.OrgID).
		Where("wpa.business_unit_id = ?", tenantInfo.BuID).
		Where("wpa.pay_profile_id = ?", profileID).
		Where(
			"wpa.effective_to IS NULL OR wpa.effective_to > extract(epoch from current_timestamp)::bigint",
		).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("wpa.effective_from DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pay assignments for profile: %w", err)
	}
	return items, nil
}

func (r *workerPayAssignmentRepository) ListOverlapping(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) ([]*driverpay.WorkerPayAssignment, error) {
	items := make([]*driverpay.WorkerPayAssignment, 0)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("wpa.organization_id = ?", entity.OrganizationID).
		Where("wpa.business_unit_id = ?", entity.BusinessUnitID).
		Where("wpa.worker_id = ?", entity.WorkerID)
	if !entity.ID.IsNil() {
		query = query.Where("wpa.id != ?", entity.ID)
	}
	if entity.EffectiveTo != nil {
		query = query.Where(
			"wpa.effective_from < ? AND (wpa.effective_to IS NULL OR wpa.effective_to > ?)",
			*entity.EffectiveTo,
			entity.EffectiveFrom,
		)
	} else {
		query = query.Where(
			"wpa.effective_to IS NULL OR wpa.effective_to > ?",
			entity.EffectiveFrom,
		)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list overlapping pay assignments: %w", err)
	}
	return items, nil
}

func (r *workerPayAssignmentRepository) Create(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) (*driverpay.WorkerPayAssignment, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("wpa_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create worker pay assignment: %w", err)
	}
	return entity, nil
}

func (r *workerPayAssignmentRepository) Update(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) (*driverpay.WorkerPayAssignment, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("pay_profile_id = ?", entity.PayProfileID).
		Set("effective_from = ?", entity.EffectiveFrom).
		Set("effective_to = ?", entity.EffectiveTo).
		Set("split_percent = ?", entity.SplitPercent).
		Set("rate_overrides = ?", entity.RateOverrides).
		Set("notes = ?", entity.Notes).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update worker pay assignment: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "WorkerPayAssignment", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *workerPayAssignmentRepository) Delete(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	res, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*driverpay.WorkerPayAssignment)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete worker pay assignment: %w", err)
	}
	return dberror.CheckRowsAffected(res, "WorkerPayAssignment", id.String())
}
