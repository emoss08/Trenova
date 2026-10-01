package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveServiceFailureReasonCodeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.ServiceFailureReasonCodeSvc.Get(
				ctx,
				repositories.GetServiceFailureReasonCodeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, serviceFailureReasonCodeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ServiceFailureReasonCodeSvc.SelectOptions(
		ctx,
		&repositories.ServiceFailureReasonCodeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		serviceFailureReasonCodeSelectOptionItem,
	)
}

func serviceFailureReasonCodeSelectOptionItem(
	entity *servicefailure.ReasonCode,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		serviceFailureReasonCodeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func serviceFailureReasonCodeSelectOption(
	entity *servicefailure.ReasonCode,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Label),
		Meta: map[string]any{
			"code":  entity.Code,
			"label": entity.Label,
		},
	}
}
