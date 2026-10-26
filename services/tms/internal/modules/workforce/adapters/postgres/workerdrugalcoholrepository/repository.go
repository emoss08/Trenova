package workerdrugalcoholrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultTestPageSize  = 200
	defaultDrawPageSize  = 50
	defaultSweepPageSize = 500
)

var testErrors = dberror.Mapping{Entity: "WorkerDOTTest"}

var violationErrors = dberror.Mapping{Entity: "WorkerDOTViolation"}

var queryErrors = dberror.Mapping{Entity: "WorkerClearinghouseQuery"}

var poolErrors = dberror.Mapping{Entity: "DOTRandomPool"}

var drawErrors = dberror.Mapping{Entity: "DOTRandomDraw"}

var drawEntryErrors = dberror.Mapping{Entity: "DOTRandomDrawEntry"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.WorkerDrugAlcoholRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-drug-alcohol-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func toNullable(col keel.Int64Col, value *int64) keel.Assignment {
	if value == nil {
		return col.ToNull()
	}

	return col.To(*value)
}

func testLoads(includeWorker, includeDocument, includeActors bool) []keel.Loader {
	t := workerdb.WorkerDOTTests
	loads := make([]keel.Loader, 0, 4)
	loads = append(loads, listquery.When(includeWorker, t.Worker())...)
	loads = append(loads, listquery.When(includeDocument, t.Document())...)

	return append(loads, listquery.When(includeActors, t.OrderedBy(), t.RecordedBy())...)
}

func testsQuery(req *workerports.ListWorkerDOTTestsRequest) *keel.Query[*worker.WorkerDOTTest] {
	t := workerdb.WorkerDOTTests

	return t.Where(
		keel.When(req.WorkerID.IsNotNil(), t.WorkerID.Eq(req.WorkerID)),
		keel.When(len(req.TestTypes) > 0, t.TestType.In(req.TestTypes...)),
		keel.When(len(req.Statuses) > 0, t.Status.In(req.Statuses...)),
		keel.When(req.OpenOnly, t.Status.In(
			worker.DOTTestStatusScheduled,
			worker.DOTTestStatusCollected,
			worker.DOTTestStatusAwaitingResult,
		)),
		keel.When(req.Since > 0, t.CollectedAt.Gte(req.Since)),
	).
		Load(testLoads(req.IncludeWorker, req.IncludeDocument, req.IncludeActors)...).
		Order(t.CollectedAt.Desc().NullsLast(), t.CreatedAt.Desc()).
		Limit(limitOr(req.Limit, defaultTestPageSize))
}

func (r *repository) ListTests(
	ctx context.Context,
	req *workerports.ListWorkerDOTTestsRequest,
) ([]*worker.WorkerDOTTest, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerDOTTest, error) {
		entities, err := testsQuery(req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list dot tests", zap.Error(err))
			return nil, fmt.Errorf("list dot tests: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetTestByID(
	ctx context.Context,
	req *workerports.GetWorkerDOTTestByIDRequest,
) (*worker.WorkerDOTTest, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerDOTTest, error) {
		t := workerdb.WorkerDOTTests
		entity, err := t.Where(t.ID.Eq(req.ID)).
			Load(testLoads(req.IncludeWorker, req.IncludeDocument, req.IncludeActors)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, testErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateTest(
	ctx context.Context,
	entity *worker.WorkerDOTTest,
) (*worker.WorkerDOTTest, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerDOTTest, error) {
		if err := workerdb.WorkerDOTTests.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create dot test", zap.Error(err))
			return nil, testErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateTest(
	ctx context.Context,
	entity *worker.WorkerDOTTest,
) (*worker.WorkerDOTTest, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerDOTTest, error) {
		if err := workerdb.WorkerDOTTests.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update dot test", zap.Error(err))
			return nil, testErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListViolations(
	ctx context.Context,
	req *workerports.ListWorkerDOTViolationsRequest,
) ([]*worker.WorkerDOTViolation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerDOTViolation, error) {
		v := workerdb.WorkerDOTViolations
		loads := listquery.When(req.IncludeTests, v.SourceTest(), v.RTDTest())
		loads = append(loads, listquery.When(req.IncludeWorker, v.Worker())...)

		entities, err := v.Where(
			keel.When(req.WorkerID.IsNotNil(), v.WorkerID.Eq(req.WorkerID)),
			keel.When(req.UnresolvedOnly, v.Status.Ne(worker.DOTViolationStatusResolved)),
		).
			Load(loads...).
			Order(v.OccurredAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list dot violations", zap.Error(err))
			return nil, fmt.Errorf("list dot violations: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetViolationByID(
	ctx context.Context,
	req *workerports.GetWorkerDOTViolationByIDRequest,
) (*worker.WorkerDOTViolation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerDOTViolation, error) {
		v := workerdb.WorkerDOTViolations
		entity, err := v.Where(v.ID.Eq(req.ID)).
			Load(listquery.When(req.IncludeTests, v.SourceTest(), v.RTDTest())...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, violationErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetOpenViolation(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerDOTViolation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerDOTViolation, error) {
		v := workerdb.WorkerDOTViolations
		entity, err := v.Where(
			v.WorkerID.Eq(workerID),
			v.Status.Ne(worker.DOTViolationStatusResolved),
		).
			Order(v.OccurredAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			r.l.Error("failed to read open dot violation", zap.Error(err))
			return nil, fmt.Errorf("read open dot violation: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) CreateViolation(
	ctx context.Context,
	entity *worker.WorkerDOTViolation,
) (*worker.WorkerDOTViolation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerDOTViolation, error) {
		if err := workerdb.WorkerDOTViolations.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create dot violation", zap.Error(err))
			return nil, violationErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateViolation(
	ctx context.Context,
	entity *worker.WorkerDOTViolation,
) (*worker.WorkerDOTViolation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerDOTViolation, error) {
		if err := workerdb.WorkerDOTViolations.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update dot violation", zap.Error(err))
			return nil, violationErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListQueries(
	ctx context.Context,
	req *workerports.ListClearinghouseQueriesRequest,
) ([]*worker.WorkerClearinghouseQuery, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*worker.WorkerClearinghouseQuery, error) {
			q := workerdb.WorkerClearinghouseQueries
			entities, err := q.Where(
				keel.When(req.WorkerID.IsNotNil(), q.WorkerID.Eq(req.WorkerID)),
				keel.When(req.PendingOnly, q.Result.Eq(worker.ClearinghouseResultPending)),
			).
				Load(listquery.When(req.IncludeWorker, q.Worker())...).
				Order(q.RequestedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list clearinghouse queries", zap.Error(err))
				return nil, fmt.Errorf("list clearinghouse queries: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) GetQueryByID(
	ctx context.Context,
	req *workerports.GetClearinghouseQueryByIDRequest,
) (*worker.WorkerClearinghouseQuery, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerClearinghouseQuery, error) {
			entity, err := workerdb.WorkerClearinghouseQueries.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, queryErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateQuery(
	ctx context.Context,
	entity *worker.WorkerClearinghouseQuery,
) (*worker.WorkerClearinghouseQuery, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerClearinghouseQuery, error) {
			err := workerdb.WorkerClearinghouseQueries.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create clearinghouse query", zap.Error(err))
				return nil, queryErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateQuery(
	ctx context.Context,
	entity *worker.WorkerClearinghouseQuery,
) (*worker.WorkerClearinghouseQuery, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerClearinghouseQuery, error) {
			err := workerdb.WorkerClearinghouseQueries.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update clearinghouse query", zap.Error(err))
				return nil, queryErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListPools(
	ctx context.Context,
	req *workerports.ListDOTRandomPoolsRequest,
) (*pagination.CursorListResult[*worker.DOTRandomPool], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*worker.DOTRandomPool], error) {
			p := workerdb.DOTRandomPools
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				p.Where(keel.When(
					req.Status != "",
					p.Status.Eq(domaintypes.Status(req.Status)),
				)),
				&listquery.Request[worker.DOTRandomPool]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list random pools", zap.Error(err))
				return nil, fmt.Errorf("list random pools: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetPoolByID(
	ctx context.Context,
	req *workerports.GetDOTRandomPoolByIDRequest,
) (*worker.DOTRandomPool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomPool, error) {
		entity, err := workerdb.DOTRandomPools.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, poolErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetDefaultPool(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*worker.DOTRandomPool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomPool, error) {
		p := workerdb.DOTRandomPools
		entity, err := p.Where(p.IsDefault.IsTrue()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			r.l.Error("failed to read default random pool", zap.Error(err))
			return nil, fmt.Errorf("read default random pool: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) PoolCodeExists(
	ctx context.Context,
	req *workerports.DOTRandomPoolCodeExistsRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		p := workerdb.DOTRandomPools
		exists, err := p.Where(
			p.Code.Lower().Eq(strings.ToLower(req.Code)),
			keel.When(req.ExcludeID.IsNotNil(), p.ID.Ne(req.ExcludeID)),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check random pool code", zap.Error(err))
			return false, fmt.Errorf("check random pool code: %w", err)
		}

		return exists, nil
	})
}

func (r *repository) CreatePool(
	ctx context.Context,
	entity *worker.DOTRandomPool,
) (*worker.DOTRandomPool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomPool, error) {
		if err := workerdb.DOTRandomPools.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create random pool", zap.Error(err))
			return nil, poolErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdatePool(
	ctx context.Context,
	entity *worker.DOTRandomPool,
) (*worker.DOTRandomPool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomPool, error) {
		if err := workerdb.DOTRandomPools.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update random pool", zap.Error(err))
			return nil, poolErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ClearDefaultPool(
	ctx context.Context,
	_ pagination.TenantInfo,
	exceptID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := workerdb.DOTRandomPools
		_, err := p.Where(
			p.IsDefault.IsTrue(),
			keel.When(exceptID.IsNotNil(), p.ID.Ne(exceptID)),
		).Update(ctx, r.db.Keel(ctx), p.IsDefault.To(false), p.Version.Increment(0))
		if err != nil {
			r.l.Error("failed to clear default random pool", zap.Error(err))
			return fmt.Errorf("clear default random pool: %w", err)
		}

		return nil
	})
}

func (r *repository) ListDraws(
	ctx context.Context,
	req *workerports.ListDOTRandomDrawsRequest,
) ([]*worker.DOTRandomDraw, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.DOTRandomDraw, error) {
		d := workerdb.DOTRandomDraws
		loads := listquery.When(req.IncludePool, d.Pool())
		loads = append(loads, listquery.When(req.IncludeActors, d.DrawnBy())...)

		entities, err := d.Where(keel.When(req.PoolID.IsNotNil(), d.PoolID.Eq(req.PoolID))).
			Load(loads...).
			Order(d.DrawnAt.Desc()).
			Limit(limitOr(req.Limit, defaultDrawPageSize)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list random draws", zap.Error(err))
			return nil, fmt.Errorf("list random draws: %w", err)
		}

		return entities, nil
	})
}

func drawLoads(req *workerports.GetDOTRandomDrawByIDRequest) []keel.Loader {
	d := workerdb.DOTRandomDraws
	loads := listquery.When(req.IncludePool, d.Pool())
	if !req.IncludeEntries {
		return loads
	}

	e := d.Entries()

	return append(loads, e.Order(e.Rank.Asc()).
		Load(listquery.When(req.IncludeWorkers, e.Worker())...))
}

func (r *repository) GetDrawByID(
	ctx context.Context,
	req *workerports.GetDOTRandomDrawByIDRequest,
) (*worker.DOTRandomDraw, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomDraw, error) {
		d := workerdb.DOTRandomDraws
		entity, err := d.Where(d.ID.Eq(req.ID)).
			Load(drawLoads(req)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, drawErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetDrawByPeriod(
	ctx context.Context,
	req *workerports.GetDOTRandomDrawByPeriodRequest,
) (*worker.DOTRandomDraw, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomDraw, error) {
		d := workerdb.DOTRandomDraws
		entity, err := d.Where(
			d.PoolID.Eq(req.PoolID),
			d.PeriodKey.Eq(req.PeriodKey),
			d.Status.Ne(worker.RandomDrawStatusCancelled),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			r.l.Error("failed to read random draw by period", zap.Error(err))
			return nil, fmt.Errorf("read random draw by period: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) CreateDrawWithEntries(
	ctx context.Context,
	draw *worker.DOTRandomDraw,
	entries []*worker.DOTRandomDrawEntry,
) (*worker.DOTRandomDraw, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomDraw, error) {
		db := r.db.Keel(ctx)
		if err := workerdb.DOTRandomDraws.Create(ctx, db, draw); err != nil {
			r.l.Error("failed to create random draw", zap.Error(err))
			return nil, drawErrors.Map(err, draw.ID.String())
		}

		for _, entry := range entries {
			entry.DrawID = draw.ID
			entry.OrganizationID = draw.OrganizationID
			entry.BusinessUnitID = draw.BusinessUnitID
		}
		if len(entries) > 0 {
			if err := workerdb.DOTRandomDrawEntries.CreateMany(ctx, db, entries); err != nil {
				r.l.Error("failed to create random draw entries", zap.Error(err))
				return nil, drawErrors.Map(err, draw.ID.String())
			}
		}

		draw.Entries = entries

		return draw, nil
	})
}

func (r *repository) UpdateDraw(
	ctx context.Context,
	entity *worker.DOTRandomDraw,
) (*worker.DOTRandomDraw, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomDraw, error) {
		if err := workerdb.DOTRandomDraws.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update random draw", zap.Error(err))
			return nil, drawErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func poolCandidatesQuery(req *workerports.ListPoolCandidatesRequest) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	wp := workerdb.WorkerProfiles
	driverTypes := make([]worker.DriverType, 0, len(req.DriverTypes))
	for _, dt := range req.DriverTypes {
		driverTypes = append(driverTypes, worker.DriverType(dt))
	}

	return w.Join(wp, wp.WorkerID.EqCol(w.ID)).
		Pick(w.ID).
		Where(
			w.Status.Eq(domaintypes.StatusActive),
			wp.DrugAlcoholStatus.Ne(worker.DrugAlcoholProhibited),
			keel.When(len(driverTypes) > 0, w.DriverType.In(driverTypes...)),
		).
		Order(w.ID.Asc())
}

func (r *repository) ListPoolCandidates(
	ctx context.Context,
	req *workerports.ListPoolCandidatesRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), poolCandidatesQuery(req))
		if err != nil {
			r.l.Error("failed to list pool candidates", zap.Error(err))
			return nil, fmt.Errorf("list pool candidates: %w", err)
		}

		return ids, nil
	})
}

func drawEntriesQuery(
	req *workerports.ListDOTRandomDrawEntriesRequest,
) *keel.Query[*worker.DOTRandomDrawEntry] {
	e := workerdb.DOTRandomDrawEntries
	loads := listquery.When(req.IncludeWorker, e.Worker())
	loads = append(loads, listquery.When(req.IncludeDraw, e.Draw())...)

	return e.Where(
		keel.When(req.DrawID.IsNotNil(), e.DrawID.Eq(req.DrawID)),
		keel.When(req.WorkerID.IsNotNil(), e.WorkerID.Eq(req.WorkerID)),
		keel.When(req.OutstandingOnly, e.Status.In(
			worker.RandomEntrySelected,
			worker.RandomEntryNotified,
		)),
	).
		Load(loads...).
		Order(e.Rank.Asc())
}

func (r *repository) ListDrawEntries(
	ctx context.Context,
	req *workerports.ListDOTRandomDrawEntriesRequest,
) ([]*worker.DOTRandomDrawEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.DOTRandomDrawEntry, error) {
		entities, err := drawEntriesQuery(req).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list draw entries", zap.Error(err))
			return nil, fmt.Errorf("list draw entries: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetDrawEntryByID(
	ctx context.Context,
	req *workerports.GetDOTRandomDrawEntryByIDRequest,
) (*worker.DOTRandomDrawEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomDrawEntry, error) {
		entity, err := workerdb.DOTRandomDrawEntries.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, drawEntryErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateDrawEntry(
	ctx context.Context,
	entity *worker.DOTRandomDrawEntry,
) (*worker.DOTRandomDrawEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.DOTRandomDrawEntry, error) {
		if err := workerdb.DOTRandomDrawEntries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update draw entry", zap.Error(err))
			return nil, drawEntryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateProfileRollup(
	ctx context.Context,
	req *workerports.UpdateDrugAlcoholRollupRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		wp := workerdb.WorkerProfiles
		_, err := wp.Where(wp.WorkerID.Eq(req.WorkerID)).Update(
			ctx,
			r.db.Keel(ctx),
			wp.DrugAlcoholStatus.To(req.Rollup.Status),
			wp.ReturnToDutyStatus.To(req.Rollup.ReturnToDuty),
			toNullable(wp.LastClearinghouseQueryAt, req.Rollup.LastClearinghouseQueryAt),
			toNullable(wp.NextClearinghouseQueryDue, req.Rollup.NextClearinghouseQueryDue),
			wp.Version.Increment(0),
		)
		if err != nil {
			r.l.Error("failed to update drug and alcohol rollup", zap.Error(err))
			return fmt.Errorf("update drug and alcohol rollup: %w", err)
		}

		return nil
	})
}

func clearinghouseDueQuery(
	req *workerports.ListWorkersWithClearinghouseDueRequest,
) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	wp := workerdb.WorkerProfiles

	return w.AllTenants().
		Join(wp, keel.And(
			wp.WorkerID.EqCol(w.ID),
			wp.OrganizationID.EqCol(w.OrganizationID),
			wp.BusinessUnitID.EqCol(w.BusinessUnitID),
		)).
		Pick(w.ID, w.OrganizationID, w.BusinessUnitID).
		Where(
			w.Status.Eq(domaintypes.StatusActive),
			keel.Or(
				wp.NextClearinghouseQueryDue.Lte(req.Until),
				wp.NextClearinghouseQueryDue.IsNull(),
			),
		).
		Order(w.ID.Asc()).
		Limit(limitOr(req.Limit, defaultSweepPageSize))
}

func (r *repository) ListWorkersWithClearinghouseDue(
	ctx context.Context,
	req *workerports.ListWorkersWithClearinghouseDueRequest,
) ([]workerports.WorkerTenantRef, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list workers due a clearinghouse query across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.WorkerTenantRef, error) {
		refs, err := keel.Into[workerports.WorkerTenantRef](
			ctx,
			r.db.Keel(ctx),
			clearinghouseDueQuery(req),
		)
		if err != nil {
			r.l.Error("failed to list workers with clearinghouse due", zap.Error(err))
			return nil, fmt.Errorf("list workers with clearinghouse due: %w", err)
		}

		return refs, nil
	})
}
