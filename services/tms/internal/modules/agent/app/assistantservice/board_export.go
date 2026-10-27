package assistantservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/docrender"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (s *Service) boardDocument(
	ctx context.Context,
	req agentports.GetThreadRequest,
	artifactID pulid.ID,
) (*docrender.Document, bool, error) {
	if s.boards == nil || s.artifacts == nil {
		return nil, false, nil
	}
	artifact, err := s.artifacts.GetByID(ctx, assistantartifactports.GetArtifactRequest{
		ID:         artifactID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil || artifact.Kind != assistantartifact.KindBoard {
		return nil, false, nil
	}

	card, err := s.boards.Card(ctx, &agentports.BoardCardRequest{
		Thread:     req,
		ArtifactID: artifactID,
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    req.UserID,
			UserID:         req.UserID,
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, true, err
	}
	_, body, _ := strings.Cut(card.Markdown, "\n")

	return &docrender.Document{
		Title:  card.Title,
		Kicker: "Board",
		Byline: strings.Join(stringutils.NonEmptyStrings(
			card.Author,
			"answered "+time.Unix(card.AnsweredAt, 0).UTC().Format("Jan 2, 2006"),
		), " · "),
		Body: strings.TrimSpace(body),
	}, true, nil
}
