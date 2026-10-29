package agentselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	*resolvercore.Core
	AiProviderService aiproviderports.AIProviderService
}

func (r *Sources) AIProvider(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if _, err := r.RequirePermission(
		ctx,
		permission.ResourceAIProvider,
		permission.OpRead,
	); err != nil {
		return nil, err
	}

	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.AiProviderService.GetByID(
				ctx,
				aiproviderports.GetAIProviderByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				if errortypes.IsNotFoundError(err) {
					continue
				}
				return nil, err
			}
			items = append(items, aiProviderSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.AiProviderService.SelectOptions(
		ctx,
		&aiproviderports.AIProviderSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			Task:               aitask.TaskAssistantChat,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		aiProviderSelectOptionItem,
	)
}

func aiProviderSelectOptionItem(entity *aiprovider.Provider) selectoption.Item {
	return selectoption.ItemFor(
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
