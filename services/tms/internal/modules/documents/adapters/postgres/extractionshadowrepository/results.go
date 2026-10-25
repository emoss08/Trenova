package extractionshadowrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionshadow/extractionshadowdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	defaultScoredLimit = 2000
	maxScoredLimit     = 10000
	defaultPurgeLimit  = 1000
	maxPurgeLimit      = 10000
)

var resultErrors = dberror.Mapping{Entity: "ExtractionShadowResult"}

type resultRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewResults(p Params) extractionshadowports.ExtractionShadowResultRepository {
	return &resultRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionshadow-result-repository"),
	}
}

func (r *resultRepository) Create(
	ctx context.Context,
	entity *extractionshadow.ShadowResult,
) (*extractionshadow.ShadowResult, bool, error) {
	return dbtx.Write2(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionshadow.ShadowResult, bool, error) {
			sr := extractionshadowdb.ShadowResults
			db := r.db.Keel(ctx)

			inserted, err := sr.CreateOrSkip(
				ctx,
				db,
				[]*extractionshadow.ShadowResult{entity},
				sr.OrganizationID,
				sr.BusinessUnitID,
				sr.DocumentID,
				sr.ExtractedAt,
			)
			if err != nil {
				r.l.Error("failed to create extraction shadow result",
					zap.String("documentId", entity.DocumentID.String()),
					zap.Error(err),
				)

				return nil, false, fmt.Errorf("create extraction shadow result: %w", err)
			}
			if inserted > 0 {
				return entity, true, nil
			}

			existing, err := byExtraction(entity.DocumentID, entity.ExtractedAt).One(ctx, db)
			if err != nil {
				return nil, false, resultErrors.Map(err, entity.DocumentID.String())
			}

			return existing, false, nil
		},
	)
}

func (r *resultRepository) GetByID(
	ctx context.Context,
	req extractionshadowports.GetExtractionShadowResultRequest,
) (*extractionshadow.ShadowResult, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowResult, error) {
		entity, err := extractionshadowdb.ShadowResults.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, resultErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *resultRepository) GetByExtraction(
	ctx context.Context,
	req extractionshadowports.GetExtractionShadowResultByExtractionRequest,
) (*extractionshadow.ShadowResult, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowResult, error) {
		entity, err := byExtraction(req.DocumentID, req.ExtractedAt).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, resultErrors.Map(err, req.DocumentID.String())
		}

		return entity, nil
	})
}

func byExtraction(
	documentID pulid.ID,
	extractedAt int64,
) *keel.Query[extractionshadow.ShadowResult] {
	sr := extractionshadowdb.ShadowResults

	return sr.Where(sr.DocumentID.Eq(documentID), sr.ExtractedAt.Eq(extractedAt))
}

func (r *resultRepository) Save(
	ctx context.Context,
	entity *extractionshadow.ShadowResult,
) (*extractionshadow.ShadowResult, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowResult, error) {
		sr := extractionshadowdb.ShadowResults
		if err := sr.SaveExcept(
			ctx,
			r.db.Keel(ctx),
			entity,
			sr.DocumentID,
			sr.ExtractedAt,
		); err != nil {
			r.l.Error("failed to save extraction shadow result", zap.Error(err))

			return nil, resultErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *resultRepository) CountCreatedSince(
	ctx context.Context,
	_ pagination.TenantInfo,
	since int64,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		sr := extractionshadowdb.ShadowResults
		count, err := sr.Where(sr.CreatedAt.Gte(since)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count extraction shadow results", zap.Error(err))

			return 0, fmt.Errorf("count extraction shadow results: %w", err)
		}

		return int(count), nil
	})
}

func (r *resultRepository) ListScored(
	ctx context.Context,
	req *extractionshadowports.ListScoredExtractionShadowResultsRequest,
) ([]*extractionshadow.ShadowResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*extractionshadow.ShadowResult, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultScoredLimit
			}
			limit = min(limit, maxScoredLimit)

			sr := extractionshadowdb.ShadowResults
			entities, err := sr.Pick(
				sr.ID,
				sr.ProviderID,
				sr.ProviderName,
				sr.ServedModel,
				sr.Verdict,
				sr.FieldResults,
				sr.ScoredCount,
				sr.CorrectCount,
				sr.CorrectedCount,
				sr.MissedCount,
				sr.BaselineFieldResults,
				sr.BaselineScoredCount,
				sr.BaselineCorrectCount,
				sr.BaselineCorrectedCount,
				sr.BaselineMissedCount,
				sr.ScoredAt,
			).
				Where(
					sr.Status.Eq(extractionshadow.ResultStatusCompleted),
					sr.ScoredAt.Gte(req.Since),
					keel.When(req.ProviderID.IsNotNil(), sr.ProviderID.Eq(req.ProviderID)),
				).
				Order(sr.ScoredAt.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list scored extraction shadow results", zap.Error(err))

				return nil, fmt.Errorf("list scored extraction shadow results: %w", err)
			}
			if entities == nil {
				entities = []*extractionshadow.ShadowResult{}
			}

			return entities, nil
		},
	)
}

func (r *resultRepository) TotalsByStatus(
	ctx context.Context,
	req extractionshadowports.TotalExtractionShadowResultsRequest,
) ([]extractionshadowports.ExtractionShadowStatusTotal, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]extractionshadowports.ExtractionShadowStatusTotal, error) {
			sr := extractionshadowdb.ShadowResults
			timed := sr.LatencyMs.Gt(0)
			totals, err := keel.Into[extractionshadowports.ExtractionShadowStatusTotal](
				ctx,
				r.db.Keel(ctx),
				sr.Pick(
					sr.Status,
					keel.Count(),
					keel.Raw("COALESCE(SUM(?), 0)", sr.CostUSD),
					keel.Raw("COALESCE(SUM(?) FILTER (WHERE ?), 0)", sr.LatencyMs, timed),
					keel.Count().Filter(timed),
				).
					Where(
						sr.CreatedAt.Gte(req.Since),
						keel.When(req.ProviderID.IsNotNil(), sr.ProviderID.Eq(req.ProviderID)),
					).
					Group(sr.Status),
			)
			if err != nil {
				r.l.Error("failed to total extraction shadow results", zap.Error(err))

				return nil, fmt.Errorf("total extraction shadow results: %w", err)
			}
			if totals == nil {
				totals = []extractionshadowports.ExtractionShadowStatusTotal{}
			}

			return totals, nil
		},
	)
}

func (r *resultRepository) ListConnection(
	ctx context.Context,
	req *extractionshadowports.ListExtractionShadowResultConnectionRequest,
) (*pagination.CursorListResult[*extractionshadow.ShadowResult], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*extractionshadow.ShadowResult], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				extractionshadowdb.ShadowResults.Query(),
				&listquery.Request[extractionshadow.ShadowResult]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list extraction shadow results", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func (r *resultRepository) PurgeBefore(
	ctx context.Context,
	req extractionshadowports.PurgeExtractionShadowResultsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultPurgeLimit
		}
		limit = min(limit, maxPurgeLimit)

		sr := extractionshadowdb.ShadowResults
		expired := sr.As("expired")
		purged, err := sr.Where(sr.ID.InQuery(expired.
			Pick(expired.ID).
			Where(expired.CreatedAt.Lt(req.Before)).
			Order(expired.CreatedAt.Asc()).
			Limit(limit),
		)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to purge extraction shadow results", zap.Error(err))

			return 0, fmt.Errorf("purge extraction shadow results: %w", err)
		}

		return purged, nil
	})
}
