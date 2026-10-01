package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

func (r *Resolver) resolveGLAccountSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.GlAccountRepo.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
			TenantInfo:   req.TenantInfo,
			GLAccountIDs: req.IDs,
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

	result, err := r.GlAccountRepo.SelectOptions(
		ctx,
		&repositories.GLAccountSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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
