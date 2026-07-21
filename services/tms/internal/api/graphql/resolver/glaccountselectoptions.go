package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

func (r *Resolver) resolveGLAccountSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.glAccountRepo.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
			TenantInfo:   req.tenantInfo,
			GLAccountIDs: req.ids,
		})
		if err != nil {
			return nil, err
		}

		items := make([]selectOptionConnectionItem, 0, len(entities))
		for _, entity := range entities {
			items = append(items, glAccountSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.glAccountRepo.SelectOptions(
		ctx,
		&repositories.GLAccountSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		glAccountSelectOptionItem,
	)
}

func glAccountSelectOptionItem(entity *glaccount.GLAccount) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		glAccountSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func glAccountSelectOption(entity *glaccount.GLAccount) *gqlmodel.SelectOption {
	description := entity.Name
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.AccountCode,
		Description: &description,
	}
}
