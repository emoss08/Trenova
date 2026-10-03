package assistanthandoffservice

import (
	"context"
	"maps"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

// handoffCallPrefix keys a copied artifact to the one it copies, so handing
// off twice into the same conversation could never make a second copy.
const handoffCallPrefix = "handoff:"

func (s *Service) pinnedArtifacts(
	ctx context.Context,
	origin *conversation.Thread,
	tenant pagination.TenantInfo,
) ([]*assistantartifact.Artifact, error) {
	if s.artifacts == nil {
		return nil, nil
	}
	page, err := s.artifacts.ListPage(ctx, repositories.ListArtifactsRequest{
		ThreadID:   origin.ID,
		TenantInfo: tenant,
		Limit:      artifactScan,
		PinnedOnly: true,
	})
	if err != nil {
		return nil, err
	}

	return Carried(latestVersions(page.Artifacts)), nil
}

// latestVersions keeps the newest version of each lineage, in the order the
// page listed the lineages: a hand-off carries what the artifact says now,
// not its history.
func latestVersions(artifacts []*assistantartifact.Artifact) []*assistantartifact.Artifact {
	newest := make(map[string]int, len(artifacts))
	out := make([]*assistantartifact.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		key := artifact.LineageID.String()
		if artifact.LineageID.IsNil() {
			key = artifact.ID.String()
		}
		at, seen := newest[key]
		switch {
		case !seen:
			newest[key] = len(out)
			out = append(out, artifact)
		case artifact.LineageSeq > out[at].LineageSeq:
			out[at] = artifact
		}
	}

	return out
}

// Carried are the pinned artifacts a hand-off copies, at most
// MaxHandoffArtifacts. A draft, a decision request or a plan is a view over a
// decision that belongs to the conversation it was made in, so it stays
// there: deciding it from another conversation would split where the
// decision lives.
func Carried(artifacts []*assistantartifact.Artifact) []*assistantartifact.Artifact {
	out := make([]*assistantartifact.Artifact, 0, conversation.MaxHandoffArtifacts)
	for _, artifact := range artifacts {
		if artifact == nil || !artifact.Pinned ||
			artifact.ProposalID.IsNotNil() || artifact.PlanID.IsNotNil() {
			continue
		}
		out = append(out, artifact)
		if len(out) == conversation.MaxHandoffArtifacts {
			break
		}
	}

	return out
}

// copyArtifacts copies the pinned artifacts into the new conversation, pinned
// there too. One that fails to copy is left out and logged rather than
// failing the hand-off: the summary still names what the person was looking
// at.
func (s *Service) copyArtifacts(
	ctx context.Context,
	tenant pagination.TenantInfo,
	thread *conversation.Thread,
	pinned []*assistantartifact.Artifact,
) []conversation.HandoffArtifact {
	out := make([]conversation.HandoffArtifact, 0, len(pinned))
	for _, source := range pinned {
		copied, err := s.artifacts.Upsert(ctx, &assistantartifact.Artifact{
			OrganizationID:   tenant.OrgID,
			BusinessUnitID:   tenant.BuID,
			ThreadID:         thread.ID,
			Kind:             source.Kind,
			Status:           source.Status,
			Title:            source.Title,
			Payload:          maps.Clone(source.Payload),
			SourceToolCallID: handoffCallPrefix + source.ID.String(),
			Pinned:           true,
		})
		if err != nil {
			s.l.Warn("could not carry a pinned artifact over a hand-off",
				zap.String("artifact", source.ID.String()), zap.Error(err))
			continue
		}
		out = append(out, conversation.HandoffArtifact{
			ID:       copied.ID,
			SourceID: source.ID,
			Title:    source.Title,
			Kind:     string(source.Kind),
		})
	}

	return out
}
