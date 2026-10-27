package extractionevalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval/extractionevaldb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	defaultPurgeLimit = 500
	maxPurgeLimit     = 5000
	maxRecentRuns     = 50
)

var runErrors = dberror.Mapping{Entity: "ExtractionEvalRun"}

type runRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRuns(p Params) extractionevalports.ExtractionEvalRunRepository {
	return &runRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractioneval-run-repository"),
	}
}

func (r *runRepository) Create(
	ctx context.Context,
	run *extractioneval.ExtractionRun,
	results []*extractioneval.ExtractionResult,
) (*extractioneval.ExtractionRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractioneval.ExtractionRun, error) {
		db := r.db.Keel(ctx)
		if err := extractionevaldb.ExtractionRuns.Create(ctx, db, run); err != nil {
			r.l.Error("failed to create extraction eval run", zap.Error(err))
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, errortypes.NewBusinessError(
					"An extraction evaluation is already running; wait for it to finish or cancel it",
				).WithInternal(err)
			}

			return nil, fmt.Errorf("create extraction eval run: %w", err)
		}

		if len(results) == 0 {
			return run, nil
		}

		for _, result := range results {
			result.RunID = run.ID
			result.OrganizationID = run.OrganizationID
			result.BusinessUnitID = run.BusinessUnitID
		}

		if err := extractionevaldb.ExtractionResults.CreateMany(ctx, db, results); err != nil {
			r.l.Error("failed to create extraction eval run", zap.Error(err))

			return nil, fmt.Errorf("create extraction eval results: %w", err)
		}

		return run, nil
	})
}

func (r *runRepository) GetByID(
	ctx context.Context,
	req extractionevalports.GetExtractionEvalRunRequest,
) (*extractioneval.ExtractionRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractioneval.ExtractionRun, error) {
		entity, err := extractionevaldb.ExtractionRuns.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, runErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *runRepository) Update(
	ctx context.Context,
	entity *extractioneval.ExtractionRun,
) (*extractioneval.ExtractionRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractioneval.ExtractionRun, error) {
		if err := extractionevaldb.ExtractionRuns.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update extraction eval run", zap.Error(err))

			return nil, runErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *runRepository) ListConnection(
	ctx context.Context,
	req *extractionevalports.ListExtractionEvalRunConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*extractioneval.ExtractionRun], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				extractionevaldb.ExtractionRuns.Query(),
				&listquery.Request[extractioneval.ExtractionRun]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list extraction eval runs", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func (r *runRepository) ListRecentFinished(
	ctx context.Context,
	_ pagination.TenantInfo,
	task aicorrection.Task,
	limit int,
) ([]*extractioneval.ExtractionRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*extractioneval.ExtractionRun, error) {
		if limit <= 0 || limit > maxRecentRuns {
			limit = maxRecentRuns
		}

		run := extractionevaldb.ExtractionRuns
		entities, err := run.Where(
			run.Task.Eq(task),
			run.Status.In(
				extractioneval.RunStatusCompleted,
				extractioneval.RunStatusBudgetStopped,
				extractioneval.RunStatusCanceled,
			),
			run.CasesCompleted.Gt(0),
		).
			Order(run.CreatedAt.Desc(), run.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list recent extraction eval runs", zap.Error(err))

			return nil, fmt.Errorf("list recent extraction eval runs: %w", err)
		}
		if entities == nil {
			entities = []*extractioneval.ExtractionRun{}
		}

		return entities, nil
	})
}

func (r *runRepository) PurgeBefore(
	ctx context.Context,
	req extractionevalports.PurgeExtractionEvalRunsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultPurgeLimit
		}
		limit = min(limit, maxPurgeLimit)

		run := extractionevaldb.ExtractionRuns
		expired := run.As("expired")
		purged, err := run.Where(run.ID.InQuery(expired.
			Pick(expired.ID).
			Where(
				expired.CreatedAt.Lt(req.Before),
				expired.Status.NotIn(extractioneval.RunStatusQueued, extractioneval.RunStatusRunning),
			).
			Order(expired.CreatedAt.Asc()).
			Limit(limit),
		)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to purge expired extraction eval runs", zap.Error(err))

			return 0, fmt.Errorf("purge expired extraction eval runs: %w", err)
		}

		return purged, nil
	})
}
