package assistantservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

// maxTurnRunes bounds the question an artifact is labelled with: a line in
// the pane, not the question again.
const maxTurnRunes = 80

// ListThreadArtifacts reads one page of a thread's artifacts by lineage, with
// each draft's and plan's status read from the proposal or plan it views, so
// the pane shows a draft as sent the moment the decision ran.
func (s *Service) ListThreadArtifacts(
	ctx context.Context,
	req repositories.GetThreadRequest,
	opts services.ListArtifactsOptions,
) (*services.AssistantArtifactPage, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if opts.Family != "" && !opts.Family.IsValid() {
		return nil, errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "There is no artifact kind by that name",
		)
	}
	empty := &services.AssistantArtifactPage{
		Results: []services.AssistantArtifact{},
		Counts: repositories.ArtifactCounts{
			Families: map[assistantartifact.Family]int{},
		},
	}
	if s.artifacts == nil {
		return empty, nil
	}

	page, err := s.artifacts.ListPage(ctx, repositories.ListArtifactsRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		Limit:      opts.Limit,
		Cursor:     opts.Cursor,
		Query:      opts.Query,
		Family:     opts.Family,
		PinnedOnly: opts.PinnedOnly,
	})
	if err != nil {
		return nil, err
	}

	return &services.AssistantArtifactPage{
		Results:    s.presentArtifacts(ctx, req, page.Artifacts),
		Total:      page.Total,
		NextCursor: page.NextCursor,
		Counts:     page.Counts,
	}, nil
}

func (s *Service) ArtifactLineage(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifactID pulid.ID,
) ([]services.AssistantArtifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	versions, err := s.artifacts.ListLineage(ctx, repositories.LineageRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		ID:         artifactID,
	})
	if err != nil {
		return nil, err
	}

	return s.presentArtifacts(ctx, req, versions), nil
}

func (s *Service) ArtifactBySlug(
	ctx context.Context,
	req repositories.GetThreadRequest,
	slug string,
) ([]services.AssistantArtifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	slug = strings.ToLower(strings.TrimSpace(slug))
	root, err := s.artifacts.FindBySlug(ctx, repositories.SlugRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		Slug:       slug,
	})
	if err != nil {
		// An old link names an artifact by its id rather than its slug.
		id, parseErr := pulid.Parse(slug)
		if parseErr != nil {
			return nil, err
		}

		return s.ArtifactLineage(ctx, req, id)
	}

	return s.ArtifactLineage(ctx, req, root.ID)
}

// presentArtifacts is what the pane reads: statuses that follow their
// decisions, and the question of the turn each artifact came from.
func (s *Service) presentArtifacts(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifacts []*assistantartifact.Artifact,
) []services.AssistantArtifact {
	s.followDecisions(ctx, req, artifacts)

	messageIDs := make([]pulid.ID, 0, len(artifacts))
	seen := make(map[pulid.ID]bool, len(artifacts))
	for _, artifact := range artifacts {
		if !artifact.MessageID.IsNil() && !seen[artifact.MessageID] {
			seen[artifact.MessageID] = true
			messageIDs = append(messageIDs, artifact.MessageID)
		}
	}
	questions, err := s.artifacts.TurnQuestions(ctx, req.ID, req.TenantInfo, messageIDs)
	if err != nil {
		s.logger.Warn("artifact turns could not be read", zap.Error(err))
	}

	out := make([]services.AssistantArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		shown := toAssistantArtifact(artifact)
		shown.Turn = turnLabel(questions[artifact.MessageID])
		out = append(out, shown)
	}

	return out
}

// turnLabel is the first line of a question, cut to a label.
func turnLabel(question string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(question), "\n")
	line = strings.TrimSpace(line)
	if utf8.RuneCountInString(line) <= maxTurnRunes {
		return line
	}

	return strings.TrimSpace(stringutils.TruncateRunes(line, maxTurnRunes-1)) + "…"
}

// assignSlug names a new lineage for its link: its title as a slug, numbered
// when the conversation already has one by that name. A later version takes
// the slug of the lineage it continues.
func (r *artifactRecorder) assignSlug(
	artifact *assistantartifact.Artifact,
	previous *assistantartifact.Artifact,
) {
	if artifact.Slug != "" {
		return
	}
	if previous != nil {
		artifact.Slug = previous.Slug
		return
	}

	base := assistantartifact.SlugBase(artifact.Title)
	taken, err := r.repo.TakenSlugs(r.ctx, r.thread.ID, r.tenant, base)
	if err != nil {
		r.logger.Warn("artifact slugs could not be read", zap.Error(err))
		taken = map[string]bool{}
	}
	artifact.Slug = assistantartifact.UniqueSlug(base, taken)
}
