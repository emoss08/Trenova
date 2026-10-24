package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var assignmentErrors = dberror.Mapping{Entity: "WorkerPayAssignment"}

type workerPayAssignmentRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewWorkerPayAssignment(p Params) driverpayports.WorkerPayAssignmentRepository {
	return &workerPayAssignmentRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-pay-assignment-repository"),
	}
}

func (r *workerPayAssignmentRepository) GetByID(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*driverpay.WorkerPayAssignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.WorkerPayAssignment, error) {
		wpa := driverpaydb.WorkerPayAssignments
		entity, err := wpa.Where(wpa.ID.Eq(id)).
			Load(wpa.PayProfile()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, assignmentErrors.Map(err, id.String())
		}

		return entity, nil
	})
}

func (r *workerPayAssignmentRepository) GetEffectiveForWorker(
	ctx context.Context,
	req driverpayports.GetWorkerPayAssignmentRequest,
) (*driverpay.WorkerPayAssignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.WorkerPayAssignment, error) {
		wpa := driverpaydb.WorkerPayAssignments
		entity, err := wpa.Where(
			wpa.WorkerID.Eq(req.WorkerID),
			wpa.EffectiveFrom.Lte(req.AsOf),
			openAfter(wpa.EffectiveTo, req.AsOf),
		).
			Load(orderedComponents(wpa.PayProfile())).
			Order(wpa.EffectiveFrom.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, assignmentErrors.Map(err, req.WorkerID.String())
			}
			return nil, fmt.Errorf("get effective pay assignment: %w", err)
		}

		return entity, nil
	})
}

func (r *workerPayAssignmentRepository) ListForWorker(
	ctx context.Context,
	req driverpayports.ListWorkerPayAssignmentsRequest,
) ([]*driverpay.WorkerPayAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*driverpay.WorkerPayAssignment, error) {
			wpa := driverpaydb.WorkerPayAssignments
			items, err := wpa.Where(wpa.WorkerID.Eq(req.WorkerID)).
				Load(wpa.PayProfile()).
				Order(wpa.EffectiveFrom.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list worker pay assignments: %w", err)
			}

			return items, nil
		},
	)
}

func (r *workerPayAssignmentRepository) ListForProfile(
	ctx context.Context,
	_ pagination.TenantInfo,
	profileID pulid.ID,
) ([]*driverpay.WorkerPayAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*driverpay.WorkerPayAssignment, error) {
			wpa := driverpaydb.WorkerPayAssignments
			items, err := wpa.Where(
				wpa.PayProfileID.Eq(profileID),
				openAfterNow(wpa.EffectiveTo),
			).
				Load(wpa.Worker()).
				Order(wpa.EffectiveFrom.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list pay assignments for profile: %w", err)
			}

			return items, nil
		},
	)
}

func (r *workerPayAssignmentRepository) ListOverlapping(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) ([]*driverpay.WorkerPayAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*driverpay.WorkerPayAssignment, error) {
			wpa := driverpaydb.WorkerPayAssignments
			overlap := openAfter(wpa.EffectiveTo, entity.EffectiveFrom)
			if entity.EffectiveTo != nil {
				overlap = keel.And(wpa.EffectiveFrom.Lt(*entity.EffectiveTo), overlap)
			}

			items, err := wpa.Where(
				wpa.WorkerID.Eq(entity.WorkerID),
				when(!entity.ID.IsNil(), wpa.ID.Ne(entity.ID)),
				overlap,
			).All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list overlapping pay assignments: %w", err)
			}

			return items, nil
		},
	)
}

func (r *workerPayAssignmentRepository) Create(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) (*driverpay.WorkerPayAssignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.WorkerPayAssignment, error) {
		if err := driverpaydb.WorkerPayAssignments.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, assignmentErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *workerPayAssignmentRepository) Update(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) (*driverpay.WorkerPayAssignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.WorkerPayAssignment, error) {
		wpa := driverpaydb.WorkerPayAssignments
		if err := wpa.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			wpa.PayProfileID,
			wpa.EffectiveFrom,
			wpa.EffectiveTo,
			wpa.SplitPercent,
			wpa.RateOverrides,
			wpa.Notes,
		); err != nil {
			return nil, assignmentErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *workerPayAssignmentRepository) Delete(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := driverpaydb.WorkerPayAssignments.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&driverpay.WorkerPayAssignment{ID: id},
		)

		return assignmentErrors.Map(err, id.String())
	})
}
