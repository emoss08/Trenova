package assistantartifactrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit = 60
	maxListLimit     = 200
)

// ErrNoIdentity is returned when an artifact carries none of the keys an
// upsert can replace on: a tool call, a proposal or a plan.
var ErrNoIdentity = errors.New("artifact needs a source tool call, a proposal or a plan")

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AssistantArtifactRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assistantartifact-repository"),
	}
}

// conflictTarget picks the partial unique index an upsert replaces on. A
// proposal or plan artifact is one per thread however many tool calls
// mention it; anything else is one per tool call and kind.
func conflictTarget(a *assistantartifact.Artifact) (string, error) {
	cols := buncolgen.ArtifactColumns
	switch {
	case !a.ProposalID.IsNil():
		return "CONFLICT (" + cols.ThreadID.Name + ", " + cols.ProposalID.Name + ") WHERE " +
			cols.ProposalID.Name + " IS NOT NULL DO UPDATE", nil
	case !a.PlanID.IsNil():
		return "CONFLICT (" + cols.ThreadID.Name + ", " + cols.PlanID.Name + ") WHERE " +
			cols.PlanID.Name + " IS NOT NULL DO UPDATE", nil
	case a.SourceToolCallID != "":
		return "CONFLICT (" + cols.ThreadID.Name + ", " + cols.SourceToolCallID.Name + ", " +
			cols.Kind.Name + ") WHERE " + cols.SourceToolCallID.Name + " <> '' DO UPDATE", nil
	default:
		return "", ErrNoIdentity
	}
}

func (r *repository) Upsert(
	ctx context.Context,
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		target, err := conflictTarget(artifact)
		if err != nil {
			return nil, err
		}

		if _, err = buildUpsert(r.db.DBForContext(ctx), artifact, target).Exec(ctx); err != nil {
			r.l.Error("failed to upsert assistant artifact",
				zap.String("threadId", artifact.ThreadID.String()),
				zap.String("kind", string(artifact.Kind)),
				zap.Error(err))

			return nil, fmt.Errorf("upsert assistant artifact: %w", err)
		}

		return artifact, nil
	})
}

func buildUpsert(db bun.IDB, artifact *assistantartifact.Artifact, target string) *bun.InsertQuery {
	cols := buncolgen.ArtifactColumns

	return db.NewInsert().
		Model(artifact).
		On(target).
		Set(cols.MessageID.SetExcluded()).
		Set(cols.RunID.SetExcluded()).
		Set(cols.Status.SetExcluded()).
		Set(cols.Title.SetExcluded()).
		Set(cols.Payload.SetExcluded()).
		Set(cols.SourceToolCallID.SetExcluded()).
		Set(cols.LineageKey.SetExcluded()).
		Set(cols.LineageID.SetExcluded()).
		Set(cols.LineageSeq.SetExcluded()).
		Set(cols.Version.IncConflict(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*")
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetArtifactRequest,
) (*assistantartifact.Artifact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		entity := new(assistantartifact.Artifact)
		cols := buncolgen.ArtifactColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ArtifactScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "AssistantArtifact")
		}

		return entity, nil
	})
}

func (r *repository) LatestInLineage(
	ctx context.Context,
	req repositories.LatestInLineageRequest,
) (*assistantartifact.Artifact, error) {
	if req.LineageKey == "" {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		cols := buncolgen.ArtifactColumns
		entities := make([]*assistantartifact.Artifact, 0, 1)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ArtifactScopeTenant(sq, req.TenantInfo).
					Where(cols.ThreadID.Eq(), req.ThreadID).
					Where(cols.LineageKey.Eq(), req.LineageKey).
					Where(cols.SourceToolCallID.Ne(), req.ExceptToolCall)
			}).
			Order(cols.LineageSeq.OrderDesc(), cols.CreatedAt.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("read latest artifact in lineage: %w", err)
		}
		if len(entities) == 0 {
			return nil, nil
		}

		return entities[0], nil
	})
}

func (r *repository) ListByToolCalls(
	ctx context.Context,
	req *repositories.ListArtifactsByToolCallsRequest,
) ([]*assistantartifact.Artifact, error) {
	if len(req.CallIDs) == 0 {
		return []*assistantartifact.Artifact{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*assistantartifact.Artifact, error) {
		cols := buncolgen.ArtifactColumns
		entities := make([]*assistantartifact.Artifact, 0, len(req.CallIDs))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ArtifactScopeTenant(sq, req.TenantInfo).
					Where(cols.ThreadID.Eq(), req.ThreadID).
					Where(cols.SourceToolCallID.In(), bun.List(req.CallIDs))
			}).
			Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list assistant artifacts by tool call",
				zap.String("threadId", req.ThreadID.String()),
				zap.Error(err))

			return nil, fmt.Errorf("list assistant artifacts by tool call: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) Delete(ctx context.Context, req *repositories.DeleteArtifactsRequest) error {
	if len(req.IDs) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.ArtifactColumns
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*assistantartifact.Artifact)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ArtifactScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ThreadID.Eq(), req.ThreadID).
					Where(cols.ID.In(), bun.List(req.IDs))
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to delete assistant artifacts",
				zap.String("threadId", req.ThreadID.String()),
				zap.Error(err))

			return fmt.Errorf("delete assistant artifacts: %w", err)
		}

		return nil
	})
}

func (r *repository) SetPinned(
	ctx context.Context,
	req repositories.SetArtifactPinnedRequest,
) (*assistantartifact.Artifact, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		cols := buncolgen.ArtifactColumns
		root, err := r.rootOf(ctx, req.TenantInfo, req.ThreadID, req.ID)
		if err != nil {
			return nil, err
		}

		updated := make([]*assistantartifact.Artifact, 0, 1)
		_, err = r.db.DBForContext(ctx).
			NewUpdate().
			Model(&updated).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ArtifactScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ThreadID.Eq(), req.ThreadID).
					Where(rootExpr+" = ?", root)
			}).
			Set(cols.Pinned.Set(), req.Pinned).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Set(cols.Version.SetExpr("{} + 1")).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("pin assistant artifact: %w", err)
		}
		if len(updated) == 0 {
			return nil, dberror.HandleNotFoundError(sql.ErrNoRows, "AssistantArtifact")
		}

		latest := updated[0]
		for _, artifact := range updated[1:] {
			if artifact.LineageSeq > latest.LineageSeq {
				latest = artifact
			}
		}

		return latest, nil
	})
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req repositories.UpdateArtifactStatusRequest,
) (*assistantartifact.Artifact, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		cols := buncolgen.ArtifactColumns
		entity := new(assistantartifact.Artifact)

		q := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ArtifactScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Set(cols.Status.Set(), req.Status).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Set(cols.Version.SetExpr("{} + 1"))
		if req.Payload != nil {
			q = q.Set(cols.Payload.Set(), req.Payload)
		}

		res, err := q.Returning("*").Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update assistant artifact status: %w", err)
		}

		if err = dberror.CheckRowsAffected(res, "AssistantArtifact", req.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) FindByProposal(
	ctx context.Context,
	tenant pagination.TenantInfo,
	proposalID pulid.ID,
) (*assistantartifact.Artifact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		entity := new(assistantartifact.Artifact)
		cols := buncolgen.ArtifactColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ArtifactScopeTenant(sq, tenant).
					Where(cols.ProposalID.Eq(), proposalID)
			}).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "AssistantArtifact")
		}

		return entity, nil
	})
}
