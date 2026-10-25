package assistantartifactrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact/assistantartifactdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

var artifactErrors = dberror.Mapping{Entity: "AssistantArtifact"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) assistantartifactports.AssistantArtifactRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assistantartifact-repository"),
	}
}

// identity is the partial unique index an upsert replaces on and the row it
// names. A proposal or plan artifact is one per thread however many tool
// calls mention it; anything else is one per tool call and kind.
type identity struct {
	conflict keel.Conflict
	match    []keel.Pred
}

func identityOf(artifact *assistantartifact.Artifact) (identity, error) {
	a := assistantartifactdb.Artifacts
	switch {
	case !artifact.ProposalID.IsNil():
		return identity{
			conflict: keel.OnConflict(a.ThreadID, a.ProposalID).Where(a.ProposalID.IsNotNull()),
			match:    []keel.Pred{a.ProposalID.Eq(artifact.ProposalID)},
		}, nil
	case !artifact.PlanID.IsNil():
		return identity{
			conflict: keel.OnConflict(a.ThreadID, a.PlanID).Where(a.PlanID.IsNotNull()),
			match:    []keel.Pred{a.PlanID.Eq(artifact.PlanID)},
		}, nil
	case artifact.SourceToolCallID != "":
		return identity{
			conflict: keel.OnConflict(a.ThreadID, a.SourceToolCallID, a.Kind).
				Where(keel.DynOf(a.SourceToolCallID).Ne(keel.DynConst(""))),
			match: []keel.Pred{
				a.SourceToolCallID.Eq(artifact.SourceToolCallID),
				a.Kind.Eq(artifact.Kind),
			},
		}, nil
	default:
		return identity{}, ErrNoIdentity
	}
}

func (r *repository) Upsert(
	ctx context.Context,
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		id, err := identityOf(artifact)
		if err != nil {
			return nil, err
		}

		a := assistantartifactdb.Artifacts
		db := r.db.Keel(ctx)
		inserted, err := a.Upsert(ctx, db, []*assistantartifact.Artifact{artifact},
			id.conflict.Update(
				a.MessageID,
				a.RunID,
				a.Status,
				a.Title,
				a.Payload,
				a.SourceToolCallID,
				a.LineageKey,
				a.LineageID,
				a.LineageSeq,
			))
		if err != nil {
			r.l.Error("failed to upsert assistant artifact",
				zap.String("threadId", artifact.ThreadID.String()),
				zap.String("kind", string(artifact.Kind)),
				zap.Error(err))

			return nil, fmt.Errorf("upsert assistant artifact: %w", err)
		}
		if inserted[0] {
			return artifact, nil
		}

		stored, err := a.Where(append(id.match, a.ThreadID.Eq(artifact.ThreadID))...).One(ctx, db)
		if err != nil {
			return nil, artifactErrors.Map(err, artifact.ID.String())
		}

		return stored, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req assistantartifactports.GetArtifactRequest,
) (*assistantartifact.Artifact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		entity, err := assistantartifactdb.Artifacts.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, artifactErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) LatestInLineage(
	ctx context.Context,
	req assistantartifactports.LatestInLineageRequest,
) (*assistantartifact.Artifact, error) {
	if req.LineageKey == "" {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		a := assistantartifactdb.Artifacts
		entity, err := a.Where(
			a.ThreadID.Eq(req.ThreadID),
			a.LineageKey.Eq(req.LineageKey),
			a.SourceToolCallID.Ne(req.ExceptToolCall),
		).
			Order(a.LineageSeq.Desc(), a.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read latest artifact in lineage: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) ListByToolCalls(
	ctx context.Context,
	req *assistantartifactports.ListArtifactsByToolCallsRequest,
) ([]*assistantartifact.Artifact, error) {
	if len(req.CallIDs) == 0 {
		return []*assistantartifact.Artifact{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*assistantartifact.Artifact, error) {
		a := assistantartifactdb.Artifacts
		entities, err := a.Where(
			a.ThreadID.Eq(req.ThreadID),
			a.SourceToolCallID.In(req.CallIDs...),
		).
			Order(a.CreatedAt.Asc(), a.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list assistant artifacts by tool call",
				zap.String("threadId", req.ThreadID.String()),
				zap.Error(err))

			return nil, fmt.Errorf("list assistant artifacts by tool call: %w", err)
		}

		return entities, nil
	})
}

type artifactTitle struct {
	ID    pulid.ID
	Title string
}

func (r *repository) ListTitles(
	ctx context.Context,
	req *assistantartifactports.ArtifactTitlesRequest,
) (map[pulid.ID]string, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]string{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		a := assistantartifactdb.Artifacts
		rows, err := keel.Into[artifactTitle](ctx, r.db.Keel(ctx), a.
			Where(a.ThreadID.Eq(req.ThreadID), a.ID.In(req.IDs...)).
			Pick(a.ID, a.Title))
		if err != nil {
			r.l.Error("failed to list assistant artifact titles",
				zap.String("threadId", req.ThreadID.String()),
				zap.Error(err))

			return nil, fmt.Errorf("list assistant artifact titles: %w", err)
		}

		titles := make(map[pulid.ID]string, len(rows))
		for _, row := range rows {
			titles[row.ID] = row.Title
		}

		return titles, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req *assistantartifactports.DeleteArtifactsRequest,
) error {
	if len(req.IDs) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		a := assistantartifactdb.Artifacts
		if _, err := a.Where(a.ThreadID.Eq(req.ThreadID), a.ID.In(req.IDs...)).
			Delete(ctx, r.db.Keel(ctx)); err != nil {
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
	req assistantartifactports.SetArtifactPinnedRequest,
) (*assistantartifact.Artifact, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		db := r.db.Keel(ctx)
		root, err := rootOf(ctx, db, req.ThreadID, req.ID)
		if err != nil {
			return nil, err
		}

		a := assistantartifactdb.Artifacts
		updated, err := a.Where(a.ThreadID.Eq(req.ThreadID), rootIs(root)).
			UpdateReturning(ctx, db, a.Pinned.To(req.Pinned))
		if err != nil {
			return nil, fmt.Errorf("pin assistant artifact: %w", err)
		}
		if len(updated) == 0 {
			return nil, artifactErrors.Map(keel.ErrNotFound, req.ID.String())
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
	req assistantartifactports.UpdateArtifactStatusRequest,
) (*assistantartifact.Artifact, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		a := assistantartifactdb.Artifacts
		sets := []keel.Assignment{a.Status.To(req.Status)}
		if req.Payload != nil {
			sets = append(sets, a.Payload.To(req.Payload))
		}

		updated, err := a.Where(a.ID.Eq(req.ID)).UpdateReturning(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return nil, fmt.Errorf("update assistant artifact status: %w", err)
		}
		if len(updated) == 0 {
			return nil, dberror.CreateVersionMismatchError("AssistantArtifact", req.ID.String())
		}

		return updated[0], nil
	})
}

func (r *repository) FindByProposal(
	ctx context.Context,
	_ pagination.TenantInfo,
	proposalID pulid.ID,
) (*assistantartifact.Artifact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantartifact.Artifact, error) {
		a := assistantartifactdb.Artifacts
		entity, err := a.Where(a.ProposalID.Eq(proposalID)).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, artifactErrors.Map(err, proposalID.String())
		}

		return entity, nil
	})
}
