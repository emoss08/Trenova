//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package ediladingrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/lading"
	"go.uber.org/zap"
)

const (
	maxReplaysListed       = 50
	maxShadowResultsListed = 500
)

var replayErrors = dberror.Mapping{Entity: "LadingReplay"}

type replays struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewReplays(p Params) ladingports.LadingReplayRepository {
	return &replays{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-lading-replay-repository"),
	}
}

func (r *replays) CreateReplay(
	ctx context.Context,
	replay *edi.LadingReplay,
) (*edi.LadingReplay, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.LadingReplay, error) {
		if replay.Items == nil {
			replay.Items = []edi.LadingReplayItem{}
		}
		if err := edidb.LadingReplays.Create(ctx, r.db.Keel(ctx), replay); err != nil {
			return nil, replayErrors.Map(err, replay.VersionID.String())
		}
		return replay, nil
	})
}

func (r *replays) GetReplay(
	ctx context.Context,
	req ladingports.GetLadingReplayRequest,
) (*edi.LadingReplay, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.LadingReplay, error) {
		rp := edidb.LadingReplays
		replay, err := rp.Where(rp.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, replayErrors.Map(err, req.ID.String())
		}
		return replay, nil
	})
}

func (r *replays) ListReplays(
	ctx context.Context,
	req ladingports.ListLadingReplaysRequest,
) ([]*edi.LadingReplay, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.LadingReplay, error) {
		rp := edidb.LadingReplays
		limit := req.Limit
		if limit <= 0 || limit > maxReplaysListed {
			limit = maxReplaysListed
		}
		return rp.Omit(rp.Items).
			Where(rp.VersionID.Eq(req.VersionID)).
			Order(rp.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *replays) FinishReplay(
	ctx context.Context,
	replay *edi.LadingReplay,
) (*edi.LadingReplay, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.LadingReplay, error) {
		rp := edidb.LadingReplays
		if replay.Items == nil {
			replay.Items = []edi.LadingReplayItem{}
		}
		err := rp.Save(
			ctx,
			r.db.Keel(ctx),
			replay,
			rp.Status,
			rp.Summary,
			rp.Items,
			rp.Error,
			rp.FinishedAt,
		)
		if err != nil {
			return nil, replayErrors.Map(err, replay.ID.String())
		}
		return replay, nil
	})
}

func (r *replays) RecordShadowResult(ctx context.Context, result *edi.LadingShadowResult) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if result.Differences == nil {
			result.Differences = []lading.Difference{}
		}
		if result.NewOutcomes == nil {
			result.NewOutcomes = []lading.ReplayOutcome{}
		}
		if result.GoneOutcomes == nil {
			result.GoneOutcomes = []lading.ReplayOutcome{}
		}
		s := edidb.LadingShadowResults
		_, err := s.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*edi.LadingShadowResult{result},
			s.VersionID,
			s.BusinessUnitID,
			s.OrganizationID,
			s.MessageID,
		)
		return err
	})
}

func (r *replays) ListShadowResults(
	ctx context.Context,
	req ladingports.ListLadingShadowResultsRequest,
) ([]*edi.LadingShadowResult, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.LadingShadowResult, error) {
		s := edidb.LadingShadowResults
		limit := req.Limit
		if limit <= 0 || limit > maxShadowResultsListed {
			limit = maxShadowResultsListed
		}
		return s.Where(
			s.VersionID.Eq(req.VersionID),
			keel.When(req.OnlyDiffering, s.Same.Eq(false)),
		).Order(s.CreatedAt.Desc(), s.ID.Desc()).Limit(limit).List(ctx, r.db.Keel(ctx))
	})
}

func (r *replays) SummarizeShadow(
	ctx context.Context,
	req ladingports.ListLadingShadowResultsRequest,
) (*ladingports.LadingShadowSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*ladingports.LadingShadowSummary, error) {
			db := r.db.Keel(ctx)
			s := edidb.LadingShadowResults
			total, err := s.Where(s.VersionID.Eq(req.VersionID)).Count(ctx, db)
			if err != nil {
				return nil, err
			}
			same, err := s.Where(s.VersionID.Eq(req.VersionID), s.Same.Eq(true)).Count(ctx, db)
			if err != nil {
				return nil, err
			}
			failed, err := s.Where(s.VersionID.Eq(req.VersionID), s.Error.IsNotNull()).
				Count(ctx, db)
			if err != nil {
				return nil, err
			}
			return &ladingports.LadingShadowSummary{Total: total, Same: same, Failed: failed}, nil
		},
	)
}
