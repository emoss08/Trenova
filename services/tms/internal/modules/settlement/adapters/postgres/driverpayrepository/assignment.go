package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment/payassignmentdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/listquery"
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
) (*payassignment.WorkerPayAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*payassignment.WorkerPayAssignment, error) {
			wpa := payassignmentdb.WorkerPayAssignments
			entity, err := wpa.Where(wpa.ID.Eq(id)).
				Load(wpa.PayProfile()).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, assignmentErrors.Map(err, id.String())
			}

			return entity, nil
		},
	)
}

func (r *workerPayAssignmentRepository) GetEffectiveForWorker(
	ctx context.Context,
	req driverpayports.GetWorkerPayAssignmentRequest,
) (*payassignment.WorkerPayAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*payassignment.WorkerPayAssignment, error) {
			wpa := payassignmentdb.WorkerPayAssignments
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
		},
	)
}

// GetEffectiveForWorkers reads, in one query, the assignment each worker is
// paid under at AsOf, with its profile and components, keyed by worker. A
// worker with no assignment in effect is absent from the map.
func (r *workerPayAssignmentRepository) GetEffectiveForWorkers(
	ctx context.Context,
	req driverpayports.GetWorkerPayAssignmentsRequest,
) (map[pulid.ID]*payassignment.WorkerPayAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*payassignment.WorkerPayAssignment, error) {
			if len(req.WorkerIDs) == 0 {
				return map[pulid.ID]*payassignment.WorkerPayAssignment{}, nil
			}

			wpa := payassignmentdb.WorkerPayAssignments
			items, err := wpa.Where(
				wpa.WorkerID.In(req.WorkerIDs...),
				wpa.EffectiveFrom.Lte(req.AsOf),
				openAfter(wpa.EffectiveTo, req.AsOf),
			).
				Load(orderedComponents(wpa.PayProfile())).
				Order(wpa.WorkerID.Asc(), wpa.EffectiveFrom.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("get effective pay assignments: %w", err)
			}

			byWorker := make(map[pulid.ID]*payassignment.WorkerPayAssignment, len(req.WorkerIDs))
			for _, item := range items {
				if _, seen := byWorker[item.WorkerID]; !seen {
					byWorker[item.WorkerID] = item
				}
			}

			return byWorker, nil
		},
	)
}

func (r *workerPayAssignmentRepository) PageForWorker(
	ctx context.Context,
	req driverpayports.ListWorkerPayAssignmentsRequest,
) (*pagination.CursorListResult[*payassignment.WorkerPayAssignment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(
			ctx context.Context,
		) (*pagination.CursorListResult[*payassignment.WorkerPayAssignment], error) {
			wpa := payassignmentdb.WorkerPayAssignments
			desc := string(dbtype.SortDirectionDesc)
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				wpa.Where(wpa.WorkerID.Eq(req.WorkerID)),
				&listquery.Request[payassignment.WorkerPayAssignment]{
					Filter: &pagination.QueryOptions{TenantInfo: req.TenantInfo},
					Cursor: req.Cursor,
					Shape: func(
						q keel.Query[*payassignment.WorkerPayAssignment],
					) keel.Query[*payassignment.WorkerPayAssignment] {
						return q.Load(wpa.PayProfile())
					},
					Keyset: &listquery.Keyset{
						Orders: []keel.Order{wpa.EffectiveFrom.Desc(), wpa.ID.Desc()},
						Shape: []pagination.CursorSortField{
							{Field: "effectiveFrom", Direction: desc},
							{Field: "id", Direction: desc},
						},
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("page worker pay assignments: %w", err)
			}

			return result, nil
		},
	)
}

func (r *workerPayAssignmentRepository) ListForProfile(
	ctx context.Context,
	_ pagination.TenantInfo,
	profileID pulid.ID,
) ([]*payassignment.WorkerPayAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*payassignment.WorkerPayAssignment, error) {
			wpa := payassignmentdb.WorkerPayAssignments
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
	entity *payassignment.WorkerPayAssignment,
) ([]*payassignment.WorkerPayAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*payassignment.WorkerPayAssignment, error) {
			wpa := payassignmentdb.WorkerPayAssignments
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
	entity *payassignment.WorkerPayAssignment,
) (*payassignment.WorkerPayAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*payassignment.WorkerPayAssignment, error) {
			if err := payassignmentdb.WorkerPayAssignments.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				return nil, assignmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *workerPayAssignmentRepository) Update(
	ctx context.Context,
	entity *payassignment.WorkerPayAssignment,
) (*payassignment.WorkerPayAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*payassignment.WorkerPayAssignment, error) {
			wpa := payassignmentdb.WorkerPayAssignments
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
		},
	)
}

func (r *workerPayAssignmentRepository) Delete(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := payassignmentdb.WorkerPayAssignments.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&payassignment.WorkerPayAssignment{ID: id},
		)

		return assignmentErrors.Map(err, id.String())
	})
}
