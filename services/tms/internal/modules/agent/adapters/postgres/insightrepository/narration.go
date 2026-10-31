package insightrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight/insightdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"go.uber.org/zap"
)

func (r *repository) ListNarrated(
	ctx context.Context,
	req insightports.ListNarratedInsightsRequest,
) ([]*insight.Insight, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*insight.Insight, error) {
		in := insightdb.Insights
		entities, err := in.Where(
			in.DetectorKey.Eq(req.DetectorKey),
			in.Status.Eq(insight.StatusActive),
			in.Narrated.Eq(true),
		).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list narrated insights",
				zap.String("operation", "ListNarrated"),
				zap.String("detector", req.DetectorKey),
				zap.Error(err))

			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req insightports.GetInsightsByIDsRequest,
) ([]*insight.Insight, error) {
	if len(req.IDs) == 0 {
		return []*insight.Insight{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*insight.Insight, error) {
		in := insightdb.Insights
		entities, err := in.Where(in.ID.In(req.IDs...)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get insights",
				zap.String("operation", "GetByIDs"),
				zap.Int("count", len(req.IDs)),
				zap.Error(err))

			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ApplyNarrations(
	ctx context.Context,
	req insightports.ApplyInsightNarrationsRequest,
) (int, error) {
	if len(req.Narrations) == 0 {
		return 0, nil
	}

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		in := insightdb.Insights
		db := r.db.Keel(ctx)
		applied := 0
		for idx := range req.Narrations {
			narration := &req.Narrations[idx]
			count, err := in.Where(in.ID.Eq(narration.ID), in.Status.Eq(insight.StatusActive)).
				Update(ctx, db,
					in.Headline.To(narration.Headline),
					in.Narrative.To(narration.Narrative),
					in.Recommendation.To(narration.Recommendation),
					in.Narrated.To(true),
					in.ModelIdentifier.To(narration.ModelIdentifier),
					in.ProviderID.To(narration.ProviderID),
					keepInsightVersion(),
				)
			if err != nil {
				r.l.Error("failed to apply insight narration",
					zap.String("operation", "ApplyNarrations"),
					zap.String("id", narration.ID.String()),
					zap.Error(err))

				return applied, fmt.Errorf("apply insight narration: %w", err)
			}
			applied += int(count)
		}

		return applied, nil
	})
}
