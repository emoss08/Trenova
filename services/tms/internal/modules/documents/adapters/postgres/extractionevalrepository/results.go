package extractionevalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval/extractionevaldb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	defaultPendingLimit = 100
	maxPendingLimit     = 500
)

var resultErrors = dberror.Mapping{Entity: "ExtractionEvalResult"}

type resultRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewResults(p Params) extractionevalports.ExtractionEvalResultRepository {
	return &resultRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractioneval-result-repository"),
	}
}

func (r *resultRepository) GetByID(
	ctx context.Context,
	req extractionevalports.GetExtractionEvalResultRequest,
) (*extractioneval.ExtractionResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*extractioneval.ExtractionResult, error) {
			entity, err := extractionevaldb.ExtractionResults.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, resultErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *resultRepository) Save(
	ctx context.Context,
	entity *extractioneval.ExtractionResult,
) (*extractioneval.ExtractionResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*extractioneval.ExtractionResult, error) {
			res := extractionevaldb.ExtractionResults
			if err := res.SaveExcept(
				ctx,
				r.db.Keel(ctx),
				entity,
				res.RunID,
				res.CaseID,
				res.Ordinal,
			); err != nil {
				r.l.Error("failed to save extraction eval result", zap.Error(err))

				return nil, resultErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *resultRepository) ListPending(
	ctx context.Context,
	req extractionevalports.ListPendingExtractionEvalResultsRequest,
) ([]*extractioneval.ExtractionResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*extractioneval.ExtractionResult, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultPendingLimit
			}
			limit = min(limit, maxPendingLimit)

			res := extractionevaldb.ExtractionResults
			entities, err := res.Pick(
				res.ID,
				res.OrganizationID,
				res.BusinessUnitID,
				res.RunID,
				res.CaseID,
				res.Ordinal,
				res.Status,
			).
				Where(
					res.RunID.Eq(req.RunID),
					res.Status.Eq(extractioneval.ResultStatusPending),
					res.Ordinal.Gt(int64(req.AfterOrdinal)),
				).
				Order(res.Ordinal.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list pending extraction eval results", zap.Error(err))

				return nil, fmt.Errorf("list pending extraction eval results: %w", err)
			}
			if entities == nil {
				entities = []*extractioneval.ExtractionResult{}
			}

			return entities, nil
		},
	)
}

func (r *resultRepository) ListByRun(
	ctx context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
) ([]*extractioneval.ExtractionResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*extractioneval.ExtractionResult, error) {
			res := extractionevaldb.ExtractionResults
			entities, err := res.Where(res.RunID.Eq(runID)).
				Order(res.Ordinal.Asc()).
				Limit(extractioneval.MaxCaseLimit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list extraction eval results", zap.Error(err))

				return nil, fmt.Errorf("list extraction eval results: %w", err)
			}
			if entities == nil {
				entities = []*extractioneval.ExtractionResult{}
			}

			return entities, nil
		},
	)
}

func (r *resultRepository) SkipPending(
	ctx context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		res := extractionevaldb.ExtractionResults
		n, err := res.Where(
			res.RunID.Eq(runID),
			res.Status.Eq(extractioneval.ResultStatusPending),
		).Update(ctx, r.db.Keel(ctx), res.Status.To(extractioneval.ResultStatusSkipped))
		if err != nil {
			r.l.Error("failed to skip pending extraction eval results", zap.Error(err))

			return 0, fmt.Errorf("skip pending extraction eval results: %w", err)
		}

		return n, nil
	})
}

func (r *resultRepository) ListConnection(
	ctx context.Context,
	req *extractionevalports.ListExtractionEvalResultConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionResult], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*extractioneval.ExtractionResult], error) {
			res := extractionevaldb.ExtractionResults
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				res.Where(res.RunID.Eq(req.RunID)),
				&listquery.Request[extractioneval.ExtractionResult]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list extraction eval results", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}
