package workerleaverepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerteam"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultCasePageSize  = 200
	defaultEntryPageSize = 1000
)

var controlErrors = dberror.Mapping{Entity: "LeaveControl"}

var caseErrors = dberror.Mapping{
	Entity: "WorkerLeaveCase",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_worker_leave_cases_worker",
			Field:   "workerId",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_leave_cases_event",
			Field:   "employmentEventId",
			Message: "Employment event does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_leave_cases_document",
			Field:   "documentId",
			Message: "Document does not exist or belongs to a different organization",
		},
	},
}

var entryErrors = dberror.Mapping{
	Entity: "WorkerLeaveEntry",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_worker_leave_entries_day",
			Field:   "usedOn",
			Message: "That day is already recorded against this case; edit it instead",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_worker_leave_entries_worker",
			Field:   "workerId",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_leave_entries_case",
			Field:   "leaveCaseId",
			Message: "Leave case does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_leave_entries_pto",
			Field:   "ptoId",
			Message: "Time off request does not exist for this worker",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.WorkerLeaveRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-leave-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

// GetControl reads the organisation's leave settings, creating the row with the
// statutory defaults if it is missing. It never returns nil: an absent control
// would read as an entitlement of zero, which would report every worker as
// having exhausted their leave.
func (r *repository) GetControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.LeaveControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.LeaveControl, error) {
		db := r.db.Keel(ctx)
		lc := workerdb.LeaveControls
		entity, err := lc.Query().First(ctx, db)
		if err == nil {
			return entity, nil
		}
		if !dberror.IsNotFoundError(err) {
			r.l.Error("failed to read leave control", zap.Error(err))
			return nil, fmt.Errorf("read leave control: %w", err)
		}

		created := worker.DefaultLeaveControl()
		created.OrganizationID = tenantInfo.OrgID
		created.BusinessUnitID = tenantInfo.BuID
		inserted, err := lc.CreateOrSkip(
			ctx,
			db,
			[]*worker.LeaveControl{created},
			lc.OrganizationID,
			lc.BusinessUnitID,
		)
		if err != nil {
			r.l.Error("failed to seed leave control", zap.Error(err))
			return nil, fmt.Errorf("seed leave control: %w", err)
		}
		if inserted > 0 {
			return created, nil
		}

		entity, err = lc.Query().First(ctx, db)
		if err != nil {
			return nil, controlErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) UpdateControl(
	ctx context.Context,
	entity *worker.LeaveControl,
) (*worker.LeaveControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.LeaveControl, error) {
		if err := workerdb.LeaveControls.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update leave control", zap.Error(err))
			return nil, controlErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// caseFilter is the one place the case predicates live, so a count and a list
// can never disagree about what they are counting.
func caseFilter(req *workerports.ListLeaveCasesRequest) []keel.Pred {
	c := workerdb.WorkerLeaveCases

	return []keel.Pred{
		keel.When(req.WorkerID.IsNotNil(), c.WorkerID.Eq(req.WorkerID)),
		keel.When(req.OpenOnly, c.Status.In(worker.LeaveCasePending, worker.LeaveCaseApproved)),
		keel.When(req.CertificationOutstandingOnly, c.CertificationStatus.In(
			worker.CertificationRequested,
			worker.CertificationInsufficient,
			worker.CertificationOverdue,
		)),
		workerteam.Where(c.WorkerID.Col, req.ManagerIDs),
	}
}

func casesQuery(req *workerports.ListLeaveCasesRequest) *keel.Query[worker.WorkerLeaveCase] {
	c := workerdb.WorkerLeaveCases
	loads := make([]keel.Loader, 0, 3)
	loads = append(loads, listquery.When(req.IncludeWorker, c.Worker())...)
	loads = append(loads, listquery.When(req.IncludeDocument, c.Document())...)
	loads = append(loads, listquery.When(req.IncludeEntries, c.Entries())...)

	return c.Where(caseFilter(req)...).
		Load(loads...).
		Order(c.StartsAt.Desc()).
		Limit(limitOr(req.Limit, defaultCasePageSize))
}

func (r *repository) ListCases(
	ctx context.Context,
	req *workerports.ListLeaveCasesRequest,
) ([]*worker.WorkerLeaveCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerLeaveCase, error) {
		entities, err := casesQuery(req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list leave cases", zap.Error(err))
			return nil, fmt.Errorf("list leave cases: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) CountCases(
	ctx context.Context,
	req *workerports.ListLeaveCasesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		total, err := workerdb.WorkerLeaveCases.Where(caseFilter(req)...).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count leave cases", zap.Error(err))
			return 0, fmt.Errorf("count leave cases: %w", err)
		}

		return int(total), nil
	})
}

func (r *repository) GetCaseByID(
	ctx context.Context,
	req *workerports.GetLeaveCaseByIDRequest,
) (*worker.WorkerLeaveCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerLeaveCase, error) {
		c := workerdb.WorkerLeaveCases
		loads := append(
			listquery.When(req.IncludeDocument, c.Document()),
			listquery.When(req.IncludeEntries, c.Entries())...,
		)

		entity, err := c.Where(c.ID.Eq(req.ID)).Load(loads...).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, caseErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateCase(
	ctx context.Context,
	entity *worker.WorkerLeaveCase,
) (*worker.WorkerLeaveCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerLeaveCase, error) {
		if err := workerdb.WorkerLeaveCases.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create leave case", zap.Error(err))
			return nil, caseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateCase(
	ctx context.Context,
	entity *worker.WorkerLeaveCase,
) (*worker.WorkerLeaveCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerLeaveCase, error) {
		if err := workerdb.WorkerLeaveCases.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update leave case", zap.Error(err))
			return nil, caseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListEntries(
	ctx context.Context,
	req *workerports.ListLeaveEntriesRequest,
) ([]*worker.WorkerLeaveEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerLeaveEntry, error) {
		e := workerdb.WorkerLeaveEntries
		entities, err := e.Where(
			keel.When(req.WorkerID.IsNotNil(), e.WorkerID.Eq(req.WorkerID)),
			keel.When(req.LeaveCaseID.IsNotNil(), e.LeaveCaseID.Eq(req.LeaveCaseID)),
			keel.When(req.Since > 0, e.UsedOn.Gte(req.Since)),
		).
			Order(e.UsedOn.Desc()).
			Limit(limitOr(req.Limit, defaultEntryPageSize)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list leave entries", zap.Error(err))
			return nil, fmt.Errorf("list leave entries: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListEntriesByCaseIDs(
	ctx context.Context,
	req *workerports.ListLeaveEntriesByCaseIDsRequest,
) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
			if len(req.LeaveCaseIDs) == 0 {
				return map[pulid.ID][]*worker.WorkerLeaveEntry{}, nil
			}

			e := workerdb.WorkerLeaveEntries
			entities, err := e.Where(e.LeaveCaseID.In(req.LeaveCaseIDs...)).
				Order(e.UsedOn.Desc()).
				Limit(len(req.LeaveCaseIDs)*defaultEntryPageSize).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list leave entries by cases", zap.Error(err))
				return nil, fmt.Errorf("list leave entries by cases: %w", err)
			}

			return sliceutils.GroupBy(entities, func(e *worker.WorkerLeaveEntry) pulid.ID {
				return e.LeaveCaseID
			}), nil
		},
	)
}

func (r *repository) GetEntryByID(
	ctx context.Context,
	req *workerports.GetLeaveEntryByIDRequest,
) (*worker.WorkerLeaveEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerLeaveEntry, error) {
		entity, err := workerdb.WorkerLeaveEntries.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, entryErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateEntry(
	ctx context.Context,
	entity *worker.WorkerLeaveEntry,
) (*worker.WorkerLeaveEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerLeaveEntry, error) {
		if err := workerdb.WorkerLeaveEntries.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create leave entry", zap.Error(err))
			return nil, entryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateEntry(
	ctx context.Context,
	entity *worker.WorkerLeaveEntry,
) (*worker.WorkerLeaveEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerLeaveEntry, error) {
		if err := workerdb.WorkerLeaveEntries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update leave entry", zap.Error(err))
			return nil, entryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteEntry(
	ctx context.Context,
	req *workerports.GetLeaveEntryByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := workerdb.WorkerLeaveEntries.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&worker.WorkerLeaveEntry{ID: req.ID},
		)
		if err != nil {
			r.l.Error("failed to delete leave entry", zap.Error(err))
			return entryErrors.Map(err, req.ID.String())
		}

		return nil
	})
}
