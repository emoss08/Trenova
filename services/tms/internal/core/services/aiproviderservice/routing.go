package aiproviderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

func (s *Service) RoutePreview(
	ctx context.Context,
	req *services.AIProviderRoutePreviewRequest,
) ([]aiprovider.TaskRoute, error) {
	draft := &aiprovider.Provider{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
	if req.Draft.ID.IsNotNil() {
		stored, err := s.repo.GetByID(ctx, repositories.GetAIProviderByIDRequest{
			ID:         req.Draft.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		draft = stored.Redacted()
	}
	applyRoutingDraft(draft, &req.Draft)

	enabled, err := s.repo.ListEnabled(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return aiprovider.PreviewRoutes(enabled, draft), nil
}

func applyRoutingDraft(provider *aiprovider.Provider, draft *services.AIProviderRoutingDraft) {
	provider.Name = draft.Name
	provider.Kind = draft.Kind
	provider.Tasks = draft.Tasks
	provider.Priority = draft.Priority
	provider.Trusted = draft.Trusted
	provider.Enabled = draft.Enabled
	if draft.EmbeddingDimensions != nil {
		provider.EmbeddingDimensions = draft.EmbeddingDimensions
	}
}
