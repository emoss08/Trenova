package assistanttest

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StubArtifactRepo struct {
	assistantartifactports.AssistantArtifactRepository

	Upserts   []*assistantartifact.Artifact
	Listed    []*assistantartifact.Artifact
	Stored    map[pulid.ID]*assistantartifact.Artifact
	Inserted  []*assistantartifact.Artifact
	Paged     []assistantartifactports.ListArtifactsRequest
	Questions map[pulid.ID]string
}

func (r *StubArtifactRepo) GetByID(
	_ context.Context,
	req assistantartifactports.GetArtifactRequest,
) (*assistantartifact.Artifact, error) {
	artifact, ok := r.Stored[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	copied := *artifact

	return &copied, nil
}

func (r *StubArtifactRepo) Upsert(
	_ context.Context,
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	if artifact.ID.IsNil() {
		artifact.ID = pulid.MustNew("art_")
	}
	copied := *artifact
	r.Upserts = append(r.Upserts, &copied)

	return artifact, nil
}

func (r *StubArtifactRepo) LatestInLineage(
	_ context.Context,
	req assistantartifactports.LatestInLineageRequest,
) (*assistantartifact.Artifact, error) {
	var latest *assistantartifact.Artifact
	for _, artifact := range r.Upserts {
		if artifact.LineageKey == req.LineageKey &&
			artifact.SourceToolCallID != req.ExceptToolCall {
			latest = artifact
		}
	}

	return latest, nil
}

func (r *StubArtifactRepo) ListPage(
	_ context.Context,
	req assistantartifactports.ListArtifactsRequest,
) (*assistantartifactports.ArtifactPage, error) {
	r.Paged = append(r.Paged, req)

	return &assistantartifactports.ArtifactPage{Artifacts: r.Listed, Total: len(r.Listed)}, nil
}

func (r *StubArtifactRepo) TakenSlugs(
	_ context.Context,
	_ pulid.ID,
	_ pagination.TenantInfo,
	_ string,
) (map[string]bool, error) {
	taken := map[string]bool{}
	for _, artifact := range r.Upserts {
		taken[artifact.Slug] = true
	}

	return taken, nil
}

// ListLineage reads a lineage from what was stored and what was written,
// oldest first.
func (r *StubArtifactRepo) ListLineage(
	_ context.Context,
	req assistantartifactports.LineageRequest,
) ([]*assistantartifact.Artifact, error) {
	all := make([]*assistantartifact.Artifact, 0, len(r.Stored)+len(r.Upserts)+len(r.Inserted))
	for _, artifact := range r.Stored {
		all = append(all, artifact)
	}
	all = append(all, r.Upserts...)
	all = append(all, r.Inserted...)

	root := pulid.Nil
	for _, artifact := range all {
		if artifact.ID == req.ID {
			root = artifact.LineageID
			if root.IsNil() {
				root = artifact.ID
			}
		}
	}
	if root.IsNil() {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	var lineage []*assistantartifact.Artifact
	for _, artifact := range all {
		if artifact.ID == root || artifact.LineageID == root {
			lineage = append(lineage, artifact)
		}
	}
	slices.SortFunc(lineage, func(a, b *assistantartifact.Artifact) int {
		return a.LineageSeq - b.LineageSeq
	})

	return lineage, nil
}

func (r *StubArtifactRepo) InsertVersion(
	_ context.Context,
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	artifact.ID = pulid.MustNew("art_")
	r.Inserted = append(r.Inserted, artifact)

	return artifact, nil
}

func (r *StubArtifactRepo) TurnQuestions(
	_ context.Context,
	_ pulid.ID,
	_ pagination.TenantInfo,
	messageIDs []pulid.ID,
) (map[pulid.ID]string, error) {
	out := map[pulid.ID]string{}
	for _, id := range messageIDs {
		out[id] = r.Questions[id]
	}

	return out, nil
}

func Observation(name string, data any) agentruntimeports.ToolObservation {
	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: "call_" + name, Name: name},
		Data: data,
	}
}
