package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveServiceFailureReasonCodeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.serviceFailureReasonCodeSvc.Get(
				ctx,
				repositories.GetServiceFailureReasonCodeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, serviceFailureReasonCodeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.serviceFailureReasonCodeSvc.SelectOptions(
		ctx,
		&repositories.ServiceFailureReasonCodeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
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
