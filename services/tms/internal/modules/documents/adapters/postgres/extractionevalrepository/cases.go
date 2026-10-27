package extractionevalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval/extractionevaldb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultCaseLimit  = extractioneval.DefaultCaseLimit
	maxActiveCaseRead = extractioneval.MaxCaseLimit
)

var caseErrors = dberror.Mapping{Entity: "ExtractionEvalCase"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type caseRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCases(p Params) extractionevalports.ExtractionEvalCaseRepository {
	return &caseRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractioneval-case-repository"),
	}
}

func (r *caseRepository) Create(
	ctx context.Context,
	entity *extractioneval.ExtractionCase,
) (*extractioneval.ExtractionCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractioneval.ExtractionCase, error) {
		if err := extractionevaldb.ExtractionCases.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, errortypes.NewBusinessError(
					"This document is already in the evaluation set",
				).WithInternal(err)
			}
			r.l.Error("failed to create extraction eval case", zap.Error(err))

			return nil, fmt.Errorf("create extraction eval case: %w", err)
		}

		return entity, nil
	})
}

func (r *caseRepository) GetByID(
	ctx context.Context,
	req extractionevalports.GetExtractionEvalCaseRequest,
) (*extractioneval.ExtractionCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractioneval.ExtractionCase, error) {
		entity, err := extractionevaldb.ExtractionCases.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, caseErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *caseRepository) GetBySourceCorrection(
	ctx context.Context,
	_ pagination.TenantInfo,
	correctionID pulid.ID,
) (*extractioneval.ExtractionCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractioneval.ExtractionCase, error) {
		c := extractionevaldb.ExtractionCases
		entities, err := c.Where(c.SourceCorrectionID.Eq(correctionID)).
			Limit(1).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if len(entities) == 0 {
			return nil, caseErrors.Map(keel.ErrNotFound, correctionID.String())
		}

		return entities[0], nil
	})
}

func (r *caseRepository) Update(
	ctx context.Context,
	entity *extractioneval.ExtractionCase,
) (*extractioneval.ExtractionCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractioneval.ExtractionCase, error) {
		c := extractionevaldb.ExtractionCases
		if err := c.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			c.Status,
			c.Title,
			c.Notes,
			c.Expected,
			c.ExpectedFieldCount,
		); err != nil {
			r.l.Error("failed to update extraction eval case", zap.Error(err))

			return nil, caseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *caseRepository) Delete(
	ctx context.Context,
	req extractionevalports.GetExtractionEvalCaseRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		c := extractionevaldb.ExtractionCases
		n, err := c.Where(c.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete extraction eval case", zap.Error(err))

			return fmt.Errorf("delete extraction eval case: %w", err)
		}

		return dberror.CheckAffected(n, "ExtractionEvalCase", req.ID.String())
	})
}

func (r *caseRepository) ListConnection(
	ctx context.Context,
	req *extractionevalports.ListExtractionEvalCaseConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionCase], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*extractioneval.ExtractionCase], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				extractionevaldb.ExtractionCases.Query(),
				&listquery.Request[extractioneval.ExtractionCase]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list extraction eval cases", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func (r *caseRepository) ListActive(
	ctx context.Context,
	req extractionevalports.ListActiveExtractionEvalCasesRequest,
) ([]*extractioneval.ExtractionCase, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*extractioneval.ExtractionCase, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultCaseLimit
			}
			limit = min(limit, maxActiveCaseRead)

			c := extractionevaldb.ExtractionCases
			entities, err := c.Pick(c.ID, c.Title, c.OrganizationID, c.BusinessUnitID).
				Where(c.Task.Eq(req.Task), c.Status.Eq(extractioneval.CaseStatusActive)).
				Order(c.CreatedAt.Desc(), c.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list active extraction eval cases", zap.Error(err))

				return nil, fmt.Errorf("list active extraction eval cases: %w", err)
			}
			if entities == nil {
				entities = []*extractioneval.ExtractionCase{}
			}

			return entities, nil
		},
	)
}

func (r *caseRepository) Counts(
	ctx context.Context,
	req extractionevalports.CountExtractionEvalCasesRequest,
) (*extractionevalports.ExtractionEvalCaseCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionevalports.ExtractionEvalCaseCounts, error) {
			type counts struct {
				Candidate int64
				Active    int64
				Retired   int64
			}

			c := extractionevaldb.ExtractionCases
			row, err := keel.IntoOne[counts](ctx, r.db.Keel(ctx), c.
				Pick(
					keel.Count().Filter(c.Status.Eq(extractioneval.CaseStatusCandidate)),
					keel.Count().Filter(c.Status.Eq(extractioneval.CaseStatusActive)),
					keel.Count().Filter(c.Status.Eq(extractioneval.CaseStatusRetired)),
				).
				Where(c.Task.Eq(req.Task)))
			if err != nil {
				r.l.Error("failed to count extraction eval cases", zap.Error(err))

				return nil, fmt.Errorf("count extraction eval cases: %w", err)
			}

			return &extractionevalports.ExtractionEvalCaseCounts{
				Candidate: int(row.Candidate),
				Active:    int(row.Active),
				Retired:   int(row.Retired),
			}, nil
		},
	)
}
