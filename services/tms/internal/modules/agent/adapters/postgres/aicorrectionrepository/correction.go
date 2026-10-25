package aicorrectionrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection/aicorrectiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval/extractionevaldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPurgeLimit    = 1000
	maxPurgeLimit        = 10000
	defaultAccuracyLimit = 2000
	defaultTrainingLimit = 100
	maxTrainingLimit     = 500
	maxAccuracyLimit     = 10000
)

var correctionErrors = dberror.Mapping{Entity: "AICorrection"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) aicorrectionports.AICorrectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aicorrection-repository"),
	}
}

func crossing[T any](q *keel.Query[T], allTenants bool) *keel.Query[T] {
	if allTenants {
		return q.AllTenants()
	}

	return q
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *aicorrection.Correction,
) (*aicorrection.Correction, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aicorrection.Correction, error) {
		c := aicorrectiondb.Corrections
		db := r.db.Keel(ctx)
		inserted, err := c.Upsert(ctx, db, []*aicorrection.Correction{entity},
			keel.OnConflict(c.OrganizationID, c.BusinessUnitID, c.SourceType, c.SourceID).Update(
				c.Task,
				c.DocumentID,
				c.SubjectType,
				c.SubjectID,
				c.CapturedByID,
				c.DocumentKind,
				c.DocumentFingerprint,
				c.ExtractionModel,
				c.ExtractionProviderID,
				c.PredictedConfidence,
				c.Predicted,
				c.Confirmed,
				c.FieldResults,
				c.ScoredCount,
				c.CorrectCount,
				c.CorrectedCount,
				c.MissedCount,
				c.UnconfirmedCount,
				c.UnscoredCount,
				c.CapturedAt,
			),
		)
		if err != nil {
			r.l.Error("failed to upsert ai correction",
				zap.String("sourceType", entity.SourceType.String()),
				zap.String("sourceId", entity.SourceID.String()),
				zap.Error(err),
			)

			return nil, correctionErrors.Map(err, entity.SourceID.String())
		}
		if inserted[0] {
			return entity, nil
		}

		stored, err := c.Where(c.SourceType.Eq(entity.SourceType), c.SourceID.Eq(entity.SourceID)).
			One(ctx, db)
		if err != nil {
			return nil, correctionErrors.Map(err, entity.SourceID.String())
		}

		return stored, nil
	})
}

func (r *repository) PurgeBefore(
	ctx context.Context,
	req aicorrectionports.PurgeAICorrectionsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultPurgeLimit
		}
		limit = min(limit, maxPurgeLimit)

		c := aicorrectiondb.Corrections
		expired := aicorrectiondb.Corrections.As("expired")
		rows, err := c.Where(c.ID.InQuery(expired.
			Where(expired.CreatedAt.Lt(req.Before)).
			Order(expired.CreatedAt.Asc()).
			Limit(limit).
			Pick(expired.ID))).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to purge expired ai corrections", zap.Error(err))

			return 0, fmt.Errorf("purge expired ai corrections: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req aicorrectionports.GetAICorrectionRequest,
) (*aicorrection.Correction, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aicorrection.Correction, error) {
		entity, err := aicorrectiondb.Corrections.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, correctionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetLatestByDocument(
	ctx context.Context,
	req *aicorrectionports.GetLatestAICorrectionByDocumentRequest,
) (*aicorrection.Correction, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aicorrection.Correction, error) {
		c := aicorrectiondb.Corrections
		entity, err := c.Where(c.Task.Eq(req.Task), c.DocumentID.Eq(req.DocumentID)).
			Order(c.CapturedAt.Desc(), c.ID.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, correctionErrors.Map(err, req.DocumentID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *aicorrectionports.ListAICorrectionConnectionRequest,
) (*pagination.CursorListResult[*aicorrection.Correction], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*aicorrection.Correction], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				aicorrectiondb.Corrections.Query(),
				&listquery.Request[aicorrection.Correction]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list ai corrections", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) TotalsByProvider(
	ctx context.Context,
	req *aicorrectionports.TotalAICorrectionsByProviderRequest,
) ([]aicorrectionports.AICorrectionProviderTotal, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]aicorrectionports.AICorrectionProviderTotal, error) {
			c := aicorrectiondb.Corrections
			totals, err := keel.Into[aicorrectionports.AICorrectionProviderTotal](
				ctx,
				r.db.Keel(ctx),
				c.Where(
					c.Task.Eq(req.Task),
					c.CapturedAt.Gte(req.Since),
					c.ExtractionProviderID.IsNotNull(),
				).
					Group(keel.Raw("1")).
					Pick(
						keel.DynOf(c.ExtractionProviderID).Eq(req.ProviderID),
						keel.DynOf(c.ScoredCount).Sum().Coalesce(0),
						keel.DynOf(c.CorrectCount).Sum().Coalesce(0),
					),
			)
			if err != nil {
				r.l.Error("failed to total ai corrections by provider", zap.Error(err))

				return nil, fmt.Errorf("total ai corrections by provider: %w", err)
			}

			return totals, nil
		},
	)
}

const (
	firstMondayEpoch = 4 * timeutils.SecondsPerDay
	secondsPerWeek   = 7 * timeutils.SecondsPerDay
)

func (r *repository) WeeklyTotalsByProvider(
	ctx context.Context,
	req *aicorrectionports.WeeklyAICorrectionTotalsRequest,
) ([]aicorrection.WeekTotal, error) {
	return r.weeklyTotals(ctx, aicorrectiondb.Corrections.Query(), req.Task, req.Since)
}

func (r *repository) WeeklyTrainableTotalsByProvider(
	ctx context.Context,
	req *aicorrectionports.WeeklyTrainableAICorrectionTotalsRequest,
) ([]aicorrection.WeekTotal, error) {
	c := aicorrectiondb.Corrections

	return r.weeklyTotals(
		ctx,
		c.AllTenants().Where(trainingConsent(true)),
		req.Task,
		req.Since,
	)
}

func (r *repository) weeklyTotals(
	ctx context.Context,
	q *keel.Query[aicorrection.Correction],
	task aicorrection.Task,
	since int64,
) ([]aicorrection.WeekTotal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]aicorrection.WeekTotal, error) {
		c := aicorrectiondb.Corrections
		totals, err := keel.Into[aicorrection.WeekTotal](ctx, r.db.Keel(ctx), q.
			Where(
				c.Task.Eq(task),
				c.CapturedAt.Gte(since),
				c.ExtractionProviderID.IsNotNull(),
			).
			Group(keel.Raw("1"), keel.Raw("2")).
			Pick(
				c.ExtractionProviderID,
				keel.Raw("? - ((? - ?) % ?)", c.CapturedAt, c.CapturedAt,
					int64(firstMondayEpoch), int64(secondsPerWeek)),
				keel.Count(),
				keel.DynOf(c.ScoredCount).Sum().Coalesce(0),
				keel.DynOf(c.CorrectCount).Sum().Coalesce(0),
			))
		if err != nil {
			r.l.Error("failed to total ai corrections by week and provider", zap.Error(err))

			return nil, fmt.Errorf("total ai corrections by week and provider: %w", err)
		}

		return totals, nil
	})
}

func trainingConsent(allTenants bool) keel.Pred {
	c := aicorrectiondb.Corrections
	consent := tenantdb.AgentControls.As("training_consent")

	return keel.Exists(crossing(consent.Where(
		consent.OrganizationID.EqCol(c.OrganizationID),
		consent.BusinessUnitID.EqCol(c.BusinessUnitID),
		consent.AITrainingConsent.IsTrue(),
	), allTenants).Pick(keel.Raw("1")))
}

func trainable(
	task aicorrection.Task,
	capturedFrom, capturedTo int64,
	allTenants bool,
) []keel.Pred {
	c := aicorrectiondb.Corrections
	promoted := extractionevaldb.ExtractionCases.As("promoted_case")

	return []keel.Pred{
		c.Task.Eq(task),
		c.CapturedAt.Gte(capturedFrom),
		c.CapturedAt.Lt(capturedTo),
		c.DocumentID.IsNotNull(),
		c.ScoredCount.Gt(0),
		keel.NotExists(crossing(promoted.Where(
			promoted.SourceCorrectionID.EqCol(c.ID),
			promoted.OrganizationID.EqCol(c.OrganizationID),
			promoted.BusinessUnitID.EqCol(c.BusinessUnitID),
		), allTenants).Pick(keel.Raw("1"))),
	}
}

func (r *repository) CountTrainable(
	ctx context.Context,
	req *aicorrectionports.CountTrainableAICorrectionsRequest,
) (int, error) {
	capPerOrganization := req.PerOrganizationCap
	if capPerOrganization <= 0 {
		return 0, errortypes.NewValidationError(
			"perOrganizationCap",
			errortypes.ErrInvalid,
			"The per-organization cap must be positive",
		)
	}

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		c := aicorrectiondb.Corrections
		perOrganization := keel.CTE("per_organization", []string{"examples"}, c.AllTenants().
			Where(trainable(req.Task, req.CapturedFrom, req.CapturedTo, true)...).
			Where(trainingConsent(true)).
			Group(c.OrganizationID, c.BusinessUnitID).
			Pick(keel.Count()))

		total, err := keel.IntoOne[int](ctx, r.db.Keel(ctx), perOrganization.Query().
			With(perOrganization).
			Pick(keel.Raw(
				"COALESCE(SUM(LEAST(?, ?)), 0)::bigint",
				perOrganization.Col("examples"),
				capPerOrganization,
			)))
		if err != nil && !errors.Is(err, keel.ErrNotFound) {
			r.l.Error("failed to count trainable ai corrections", zap.Error(err))

			return 0, fmt.Errorf("count trainable ai corrections: %w", err)
		}

		return total, nil
	})
}

func (r *repository) ListForAccuracy(
	ctx context.Context,
	req aicorrectionports.ListAICorrectionsForAccuracyRequest,
) ([]*aicorrection.Correction, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aicorrection.Correction, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultAccuracyLimit
		}
		limit = min(limit, maxAccuracyLimit)

		c := aicorrectiondb.Corrections
		entities, err := c.
			Pick(
				c.ID,
				c.DocumentKind,
				c.ExtractionModel,
				c.ExtractionProviderID,
				c.FieldResults,
				c.ScoredCount,
				c.CorrectCount,
				c.CorrectedCount,
				c.MissedCount,
				c.UnconfirmedCount,
				c.UnscoredCount,
				c.CapturedAt,
			).
			Where(c.Task.Eq(req.Task), c.CapturedAt.Gte(req.Since)).
			Order(c.CapturedAt.Desc(), c.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list ai corrections for accuracy", zap.Error(err))

			return nil, fmt.Errorf("list ai corrections for accuracy: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListForTraining(
	ctx context.Context,
	req *aicorrectionports.ListAICorrectionsForTrainingRequest,
) ([]*aicorrection.Correction, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*aicorrection.Correction, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultTrainingLimit
		}
		limit = min(limit, maxTrainingLimit)

		c := aicorrectiondb.Corrections
		q := c.Where(trainable(req.Task, req.CapturedFrom, req.CapturedTo, false)...)
		if req.BeforeID.IsNotNil() {
			q.Where(keel.Or(
				c.CapturedAt.Lt(req.BeforeCapturedAt),
				keel.And(c.CapturedAt.Eq(req.BeforeCapturedAt), c.ID.Lt(req.BeforeID)),
			))
		}

		entities, err := q.
			Order(c.CapturedAt.Desc(), c.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list ai corrections for training", zap.Error(err))

			return nil, fmt.Errorf("list ai corrections for training: %w", err)
		}

		return entities, nil
	})
}
