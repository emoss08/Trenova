package agentprofilerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile/agentprofiledb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

const (
	defaultGuidanceList = 100
	maxGuidanceList     = 200
)

var guidanceErrors = dberror.Mapping{Entity: "Guidance"}

type guidanceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewGuidance(p Params) agentprofileports.GuidanceRepository {
	return &guidanceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-profile-guidance-repository"),
	}
}

func guidanceSubject(ref agentprofileports.SubjectRef) keel.Pred {
	g := agentprofiledb.Guidances
	person := g.UserID.IsNull()
	if ref.Subject == agentprofile.SubjectUser {
		person = g.UserID.Eq(ref.PersonID)
	}

	return keel.And(g.Subject.Eq(ref.Subject), person)
}

// ListActive is what the next version of the profile is written from, oldest
// first, so a later line reads as refining an earlier one.
func (r *guidanceRepository) ListActive(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) ([]*agentprofile.Guidance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentprofile.Guidance, error) {
		g := agentprofiledb.Guidances
		rows, err := g.Where(guidanceSubject(ref), g.Status.Eq(agentprofile.GuidanceStatusActive)).
			Order(g.CreatedAt.Asc(), g.ID.Asc()).
			Limit(agentprofile.MaxActiveGuidance).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list active agent profile guidance: %w", err)
		}

		return rows, nil
	})
}

// List is what AI Control and the Desk show: the active and the offered lines,
// and the closed ones when asked, newest first, with who gave each.
func (r *guidanceRepository) List(
	ctx context.Context,
	req *agentprofileports.ListGuidanceRequest,
) ([]*agentprofile.Guidance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentprofile.Guidance, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultGuidanceList
		}
		limit = min(limit, maxGuidanceList)

		g := agentprofiledb.Guidances
		rows, err := g.Load(g.CreatedBy()).
			Where(
				guidanceSubject(req.Ref),
				keel.When(!req.IncludeRetired, g.Status.In(openGuidanceStatuses()...)),
			).
			Order(
				keel.Asc(keel.Raw(
					"CASE ? WHEN ? THEN 0 WHEN ? THEN 1 ELSE 2 END",
					g.Status,
					string(agentprofile.GuidanceStatusActive),
					string(agentprofile.GuidanceStatusSuggested),
				)),
				g.CreatedAt.Desc(),
				g.ID.Desc(),
			).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent profile guidance: %w", err)
		}

		return rows, nil
	})
}

func (r *guidanceRepository) CountActive(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) (int, error) {
	return r.count(ctx, ref, []agentprofile.GuidanceStatus{agentprofile.GuidanceStatusActive})
}

func (r *guidanceRepository) CountOpen(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) (int, error) {
	return r.count(ctx, ref, openGuidanceStatuses())
}

func (r *guidanceRepository) count(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
	statuses []agentprofile.GuidanceStatus,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		g := agentprofiledb.Guidances
		n, err := g.Where(guidanceSubject(ref), g.Status.In(statuses...)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count agent profile guidance: %w", err)
		}

		return int(n), nil
	})
}

// openGuidanceStatuses are the lines a profile still holds: those shaping it
// and those offered and waiting.
func openGuidanceStatuses() []agentprofile.GuidanceStatus {
	return []agentprofile.GuidanceStatus{
		agentprofile.GuidanceStatusActive,
		agentprofile.GuidanceStatusSuggested,
	}
}

func (r *guidanceRepository) GetByID(
	ctx context.Context,
	req agentprofileports.GetGuidanceRequest,
) (*agentprofile.Guidance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentprofile.Guidance, error) {
		entity, err := agentprofiledb.Guidances.Get(ctx, r.db.Keel(ctx), req.ID)
		if dberror.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError("Guidance not found")
		}
		if err != nil {
			return nil, fmt.Errorf("get agent profile guidance: %w", err)
		}

		return entity, nil
	})
}

func (r *guidanceRepository) Create(
	ctx context.Context,
	entity *agentprofile.Guidance,
) (*agentprofile.Guidance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentprofile.Guidance, error) {
		if err := agentprofiledb.Guidances.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create agent profile guidance", zap.Error(err))

			return nil, fmt.Errorf("create agent profile guidance: %w", err)
		}

		return entity, nil
	})
}

// Update saves a change to a line under its version, so two people retiring
// the same line at once leave one record of who did.
func (r *guidanceRepository) Update(
	ctx context.Context,
	entity *agentprofile.Guidance,
) (*agentprofile.Guidance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentprofile.Guidance, error) {
		g := agentprofiledb.Guidances
		err := g.Save(ctx, r.db.Keel(ctx), entity, g.Status, g.RetiredByID, g.RetiredAt)
		if err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, guidanceErrors.Map(err, entity.ID.String())
			}

			return nil, fmt.Errorf("update agent profile guidance: %w", err)
		}

		return entity, nil
	})
}
