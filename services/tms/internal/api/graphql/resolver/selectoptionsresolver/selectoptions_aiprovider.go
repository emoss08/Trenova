package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Deps) resolveAIProviderSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if _, err := r.RequirePermission(ctx, permission.ResourceAIProvider, permission.OpRead); err != nil {
		return nil, err
	}

	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.AiProviderService.GetByID(ctx, repositories.GetAIProviderByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				if errortypes.IsNotFoundError(err) {
					continue
				}
				return nil, err
			}
			items = append(items, aiProviderSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.AiProviderService.SelectOptions(ctx, &repositories.AIProviderSelectOptionsRequest{
		SelectQueryRequest: req.SelectQuery,
		Task:               aiprovider.TaskAssistantChat,
	})
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		aiProviderSelectOptionItem,
	)
}

func aiProviderSelectOptionItem(entity *aiprovider.Provider) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Model),
			Meta: map[string]any{
				"model":   entity.Model,
				"kind":    string(entity.Kind),
				"enabled": entity.Enabled,
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}
