package aifeedbackrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback/aifeedbackdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultNegativeLimit = 5000
	maxNegativeLimit     = 20000
	defaultWorstLimit    = 10
	maxWorstLimit        = 1000
	defaultPurgeLimit    = 1000
	maxPurgeLimit        = 10000
)

var feedbackErrors = dberror.Mapping{Entity: "AIFeedback"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) aifeedbackports.AIFeedbackRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aifeedback-repository"),
	}
}

func targetIs(target aifeedbackports.AIFeedbackTargetRef) keel.Pred {
	f := aifeedbackdb.Feedbacks

	return keel.And(
		f.TargetType.Eq(target.TargetType),
		f.TargetID.Eq(target.TargetID),
		f.TargetPart.Eq(target.TargetPart),
	)
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *aifeedback.Feedback,
) (*aifeedback.Feedback, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aifeedback.Feedback, error) {
		f := aifeedbackdb.Feedbacks
		db := r.db.Keel(ctx)
		inserted, err := f.Upsert(ctx, db, []*aifeedback.Feedback{entity},
			keel.OnConflict(
				f.OrganizationID,
				f.BusinessUnitID,
				f.UserID,
				f.TargetType,
				f.TargetID,
				f.TargetPart,
			).Update(
				f.ThreadID,
				f.TurnID,
				f.RunID,
				f.AgentDefinitionID,
				f.DefinitionVersion,
				f.DetectorKey,
				f.Task,
				f.Model,
				f.ProviderID,
				f.PromptHash,
				f.ToolSpecHash,
				f.FingerprintSource,
				f.Rating,
				f.Reasons,
				f.Comment,
				f.TurnSnapshot,
				f.PatternKey,
			),
		)
		if err != nil {
			r.l.Error("failed to upsert ai feedback",
				zap.String("targetType", entity.TargetType.String()),
				zap.Error(err),
			)

			return nil, feedbackErrors.Map(err, entity.TargetID.String())
		}
		if inserted[0] {
			return entity, nil
		}

		stored, err := f.Where(
			f.UserID.Eq(entity.UserID),
			targetIs(aifeedbackports.AIFeedbackTargetRef{
				TargetType: entity.TargetType,
				TargetID:   entity.TargetID,
				TargetPart: entity.TargetPart,
			}),
		).One(ctx, db)
		if err != nil {
			return nil, feedbackErrors.Map(err, entity.TargetID.String())
		}

		return stored, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req aifeedbackports.DeleteAIFeedbackRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		f := aifeedbackdb.Feedbacks
		rows, err := f.Where(f.UserID.Eq(req.UserID), targetIs(req.Target)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("delete ai feedback: %w", err)
		}

		return rows > 0, nil
	})
}

func (r *repository) ListForTargets(
	ctx context.Context,
	req aifeedbackports.ListAIFeedbackForTargetsRequest,
) ([]*aifeedback.Feedback, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aifeedback.Feedback, error) {
		if len(req.Targets) == 0 {
			return []*aifeedback.Feedback{}, nil
		}

		f := aifeedbackdb.Feedbacks
		targets := make([]keel.Pred, 0, len(req.Targets))
		for _, target := range req.Targets {
			targets = append(targets, targetIs(target))
		}

		rows, err := f.Where(f.UserID.Eq(req.UserID), keel.Or(targets...)).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list ai feedback for targets", zap.Error(err))

			return nil, fmt.Errorf("list ai feedback for targets: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req aifeedbackports.ListAIFeedbackByIDsRequest,
) ([]*aifeedback.Feedback, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aifeedback.Feedback, error) {
		if len(req.IDs) == 0 {
			return []*aifeedback.Feedback{}, nil
		}

		f := aifeedbackdb.Feedbacks
		rows, err := f.Where(f.ID.In(req.IDs...)).Limit(len(req.IDs)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list ai feedback by ids: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *aifeedbackports.ListAIFeedbackConnectionRequest,
) (*pagination.CursorListResult[*aifeedback.Feedback], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*aifeedback.Feedback], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				aifeedbackdb.Feedbacks.Query(),
				&listquery.Request[aifeedback.Feedback]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list ai feedback", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func ratingCount(rating aifeedback.Rating) keel.Agg {
	return keel.Count().Filter(aifeedbackdb.Feedbacks.Rating.Eq(rating))
}

func (r *repository) DailySatisfaction(
	ctx context.Context,
	req aifeedbackports.AIFeedbackWindowRequest,
) ([]*aifeedbackports.AIFeedbackDay, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*aifeedbackports.AIFeedbackDay, error) {
			f := aifeedbackdb.Feedbacks
			days, err := keel.Into[aifeedbackports.AIFeedbackDay](ctx, r.db.Keel(ctx),
				f.Where(windowScope(&req)...).
					Group(keel.Raw("1")).
					Pick(
						keel.Raw(
							"to_char(to_timestamp(?) AT TIME ZONE ?, 'YYYY-MM-DD')",
							f.CreatedAt,
							timezoneOf(req.Timezone),
						),
						ratingCount(aifeedback.RatingPositive),
						ratingCount(aifeedback.RatingNegative),
					).
					Order(keel.Asc(keel.Raw("1"))))
			if err != nil {
				r.l.Error("failed to read ai feedback satisfaction by day", zap.Error(err))

				return nil, fmt.Errorf("ai feedback satisfaction by day: %w", err)
			}

			rows := make([]*aifeedbackports.AIFeedbackDay, 0, len(days))
			for index := range days {
				rows = append(rows, &days[index])
			}

			return rows, nil
		},
	)
}

func (r *repository) WorstRated(
	ctx context.Context,
	req aifeedbackports.AIFeedbackWindowRequest,
) ([]*aifeedbackports.AIFeedbackTargetScore, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*aifeedbackports.AIFeedbackTargetScore, error) {
			f := aifeedbackdb.Feedbacks
			limit := req.Limit
			if limit <= 0 {
				limit = defaultWorstLimit
			}
			limit = min(limit, maxWorstLimit)

			negative := ratingCount(aifeedback.RatingNegative)
			positive := ratingCount(aifeedback.RatingPositive)
			scores, err := keel.Into[aifeedbackports.AIFeedbackTargetScore](ctx, r.db.Keel(ctx),
				f.Where(windowScope(&req)...).
					Group(f.TargetType, f.TargetID, f.TargetPart).
					Having(keel.DynOf(negative).Gt(0)).
					Pick(
						f.TargetType,
						f.TargetID,
						f.TargetPart,
						positive,
						negative,
						f.CreatedAt.Max(),
						keel.Raw(
							"(ARRAY_AGG(? ORDER BY ? ASC, ? DESC))[1]",
							f.ID,
							f.Rating,
							f.CreatedAt,
						),
					).
					Order(
						keel.Desc(keel.DynOf(negative).Sub(positive)),
						keel.Desc(negative),
						keel.Desc(f.CreatedAt.Max()),
						f.TargetType.Asc(),
						f.TargetID.Asc(),
						f.TargetPart.Asc(),
					).
					Limit(limit).
					Offset(max(req.Offset, 0)))
			if err != nil {
				r.l.Error("failed to read worst-rated ai output", zap.Error(err))

				return nil, fmt.Errorf("worst-rated ai output: %w", err)
			}

			rows := make([]*aifeedbackports.AIFeedbackTargetScore, 0, len(scores))
			for index := range scores {
				rows = append(rows, &scores[index])
			}

			return rows, nil
		},
	)
}

func windowScope(req *aifeedbackports.AIFeedbackWindowRequest) []keel.Pred {
	f := aifeedbackdb.Feedbacks
	preds := []keel.Pred{f.CreatedAt.Gte(req.Since)}
	if req.AgentDefinitionID.IsNotNil() {
		return append(preds, f.AgentDefinitionID.Eq(req.AgentDefinitionID))
	}

	return append(preds, f.AgentDefinitionID.IsNotNull())
}

func (r *repository) TotalsByAgent(
	ctx context.Context,
	req aifeedbackports.AIFeedbackAgentTotalsRequest,
) ([]*aifeedbackports.AIFeedbackAgentTotals, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*aifeedbackports.AIFeedbackAgentTotals, error) {
			f := aifeedbackdb.Feedbacks
			q := f.Where(f.AgentDefinitionID.IsNotNull(), f.CreatedAt.Gte(req.Since))
			if req.Until > 0 {
				q.Where(f.CreatedAt.Lt(req.Until))
			}
			if len(req.AgentDefinitionIDs) > 0 {
				q.Where(f.AgentDefinitionID.In(req.AgentDefinitionIDs...))
			}

			totals, err := keel.Into[aifeedbackports.AIFeedbackAgentTotals](ctx, r.db.Keel(ctx), q.
				Group(f.AgentDefinitionID).
				Pick(
					f.AgentDefinitionID,
					ratingCount(aifeedback.RatingPositive),
					ratingCount(aifeedback.RatingNegative),
				))
			if err != nil {
				r.l.Error("failed to count ai feedback by agent", zap.Error(err))

				return nil, fmt.Errorf("ai feedback totals by agent: %w", err)
			}

			rows := make([]*aifeedbackports.AIFeedbackAgentTotals, 0, len(totals))
			for index := range totals {
				rows = append(rows, &totals[index])
			}

			return rows, nil
		},
	)
}

func timezoneOf(timezone string) string {
	if strings.TrimSpace(timezone) == "" {
		return "UTC"
	}

	return timezone
}

func (r *repository) ListNegativeSince(
	ctx context.Context,
	req aifeedbackports.ListNegativeAIFeedbackRequest,
) ([]*aifeedback.Feedback, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aifeedback.Feedback, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultNegativeLimit
		}
		limit = min(limit, maxNegativeLimit)

		f := aifeedbackdb.Feedbacks
		rows, err := f.
			Pick(
				f.ID,
				f.BusinessUnitID,
				f.OrganizationID,
				f.UserID,
				f.TargetType,
				f.TargetID,
				f.TargetPart,
				f.ThreadID,
				f.AgentDefinitionID,
				f.Rating,
				f.Reasons,
				f.Comment,
				f.PatternKey,
				f.CreatedAt,
			).
			Where(
				f.Rating.Eq(aifeedback.RatingNegative),
				f.AgentDefinitionID.IsNotNull(),
				f.CreatedAt.Gte(req.Since),
			).
			Order(f.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list negative ai feedback", zap.Error(err))

			return nil, fmt.Errorf("list negative ai feedback: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) PurgeBefore(
	ctx context.Context,
	req aifeedbackports.PurgeAIFeedbackRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultPurgeLimit
		}
		limit = min(limit, maxPurgeLimit)

		f := aifeedbackdb.Feedbacks
		expired := aifeedbackdb.Feedbacks.As("expired")
		rows, err := f.Where(f.ID.InQuery(expired.
			Where(expired.CreatedAt.Lt(req.Before)).
			Order(expired.CreatedAt.Asc()).
			Limit(limit).
			Pick(expired.ID))).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to purge expired ai feedback", zap.Error(err))

			return 0, fmt.Errorf("purge expired ai feedback: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) LinkEvalCase(
	ctx context.Context,
	req aifeedbackports.LinkAIFeedbackEvalCaseRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		f := aifeedbackdb.Feedbacks
		rows, err := f.Where(f.ID.Eq(req.FeedbackID)).
			Update(ctx, r.db.Keel(ctx), f.EvalCaseID.To(req.EvalCaseID))
		if err != nil {
			r.l.Error("failed to link ai feedback to an evaluation case", zap.Error(err))

			return fmt.Errorf("link ai feedback to evaluation case: %w", err)
		}

		return dberror.CheckAffected(rows, "AIFeedback", req.FeedbackID.String())
	})
}
