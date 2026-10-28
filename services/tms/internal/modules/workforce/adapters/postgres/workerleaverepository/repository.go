package workerleaverepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerteam"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave/workerleavedb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerleaveports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
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

var caseErrors = dberror.Mapping{Entity: "WorkerLeaveCase"}

var entryErrors = dberror.Mapping{Entity: "WorkerLeaveEntry"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerleaveports.WorkerLeaveRepository {
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
) (*workerleave.LeaveControl, error) {
	lc := workerleavedb.LeaveControls
	newDefault := func() *workerleave.LeaveControl {
		created := workerleave.DefaultLeaveControl()
		created.OrganizationID = tenantInfo.OrgID
		created.BusinessUnitID = tenantInfo.BuID
		return created
	}

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*workerleave.LeaveControl]{
		Find: func(ctx context.Context) (*workerleave.LeaveControl, error) {
			return lc.Cache(resultcache.Settings()...).First(ctx, r.db.Keel(ctx))
		},
		Create: func(ctx context.Context) error {
			if _, err := lc.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*workerleave.LeaveControl{newDefault()},
				lc.OrganizationID,
				lc.BusinessUnitID,
			); err != nil {
				r.l.Error("failed to seed leave control", zap.Error(err))
				return fmt.Errorf("seed leave control: %w", err)
			}

			return nil
		},
		Default: newDefault,
		MapError: func(err error) error {
			r.l.Error("failed to read leave control", zap.Error(err))
			return controlErrors.Map(err, "")
		},
	})
}

func (r *repository) UpdateControl(
	ctx context.Context,
	entity *workerleave.LeaveControl,
) (*workerleave.LeaveControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerleave.LeaveControl, error) {
		if err := workerleavedb.LeaveControls.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update leave control", zap.Error(err))
			return nil, controlErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// caseFilter is the one place the case predicates live, so a count and a list
// can never disagree about what they are counting.
func caseFilter(req *workerleaveports.ListLeaveCasesRequest) []keel.Pred {
	c := workerleavedb.WorkerLeaveCases

	return []keel.Pred{
		keel.When(req.WorkerID.IsNotNil(), c.WorkerID.Eq(req.WorkerID)),
		keel.When(
			req.OpenOnly,
			c.Status.In(workerleave.LeaveCasePending, workerleave.LeaveCaseApproved),
		),
		keel.When(req.CertificationOutstandingOnly, c.CertificationStatus.In(
			workerleave.CertificationRequested,
			workerleave.CertificationInsufficient,
			workerleave.CertificationOverdue,
		)),
		workerteam.Where(c.WorkerID.Col, req.ManagerIDs),
	}
}

func casesQuery(
	req *workerleaveports.ListLeaveCasesRequest,
) *keel.Query[*workerleave.WorkerLeaveCase] {
	c := workerleavedb.WorkerLeaveCases
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
	req *workerleaveports.ListLeaveCasesRequest,
) ([]*workerleave.WorkerLeaveCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*workerleave.WorkerLeaveCase, error) {
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
	req *workerleaveports.ListLeaveCasesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		total, err := workerleavedb.WorkerLeaveCases.Where(caseFilter(req)...).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count leave cases", zap.Error(err))
			return 0, fmt.Errorf("count leave cases: %w", err)
		}

		return int(total), nil
	})
}

func (r *repository) GetCaseByID(
	ctx context.Context,
	req *workerleaveports.GetLeaveCaseByIDRequest,
) (*workerleave.WorkerLeaveCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workerleave.WorkerLeaveCase, error) {
		c := workerleavedb.WorkerLeaveCases
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
	entity *workerleave.WorkerLeaveCase,
) (*workerleave.WorkerLeaveCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerleave.WorkerLeaveCase, error) {
		if err := workerleavedb.WorkerLeaveCases.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create leave case", zap.Error(err))
			return nil, caseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateCase(
	ctx context.Context,
	entity *workerleave.WorkerLeaveCase,
) (*workerleave.WorkerLeaveCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerleave.WorkerLeaveCase, error) {
		if err := workerleavedb.WorkerLeaveCases.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update leave case", zap.Error(err))
			return nil, caseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListEntries(
	ctx context.Context,
	req *workerleaveports.ListLeaveEntriesRequest,
) ([]*workerleave.WorkerLeaveEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*workerleave.WorkerLeaveEntry, error) {
		e := workerleavedb.WorkerLeaveEntries
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
	req *workerleaveports.ListLeaveEntriesByCaseIDsRequest,
) (map[pulid.ID][]*workerleave.WorkerLeaveEntry, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*workerleave.WorkerLeaveEntry, error) {
			if len(req.LeaveCaseIDs) == 0 {
				return map[pulid.ID][]*workerleave.WorkerLeaveEntry{}, nil
			}

			e := workerleavedb.WorkerLeaveEntries
			entities, err := e.Where(e.LeaveCaseID.In(req.LeaveCaseIDs...)).
				Order(e.UsedOn.Desc()).
				Limit(len(req.LeaveCaseIDs)*defaultEntryPageSize).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list leave entries by cases", zap.Error(err))
				return nil, fmt.Errorf("list leave entries by cases: %w", err)
			}

			return sliceutils.GroupBy(entities, func(e *workerleave.WorkerLeaveEntry) pulid.ID {
				return e.LeaveCaseID
			}), nil
		},
	)
}

func (r *repository) GetEntryByID(
	ctx context.Context,
	req *workerleaveports.GetLeaveEntryByIDRequest,
) (*workerleave.WorkerLeaveEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workerleave.WorkerLeaveEntry, error) {
		entity, err := workerleavedb.WorkerLeaveEntries.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, entryErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateEntry(
	ctx context.Context,
	entity *workerleave.WorkerLeaveEntry,
) (*workerleave.WorkerLeaveEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerleave.WorkerLeaveEntry, error) {
		if err := workerleavedb.WorkerLeaveEntries.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create leave entry", zap.Error(err))
			return nil, entryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateEntry(
	ctx context.Context,
	entity *workerleave.WorkerLeaveEntry,
) (*workerleave.WorkerLeaveEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerleave.WorkerLeaveEntry, error) {
		if err := workerleavedb.WorkerLeaveEntries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update leave entry", zap.Error(err))
			return nil, entryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteEntry(
	ctx context.Context,
	req *workerleaveports.GetLeaveEntryByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := workerleavedb.WorkerLeaveEntries.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&workerleave.WorkerLeaveEntry{ID: req.ID},
		)
		if err != nil {
			r.l.Error("failed to delete leave entry", zap.Error(err))
			return entryErrors.Map(err, req.ID.String())
		}

		return nil
	})
}
