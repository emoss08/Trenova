package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

func (r *Resolver) resolvePayCodeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.driverPayService.GetPayCode(ctx, repositories.GetPayCodeByIDRequest{
				ID:         id,
				TenantInfo: req.tenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, payCodeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.driverPayService.PayCodeSelectOptions(
		ctx,
		&repositories.PayCodeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			Direction: driverpay.PayCodeDirection(
				selectOptionStringFilter(req.filters, "direction"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		payCodeSelectOptionItem,
	)
}

func payCodeSelectOptionItem(entity *driverpay.PayCode) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Code,
			Description: stringPtr(entity.Name),
			Meta: map[string]any{
				"code":                  entity.Code,
				"name":                  entity.Name,
				"direction":             string(entity.Direction),
				"taxable":               entity.Taxable,
				"countsTowardGuarantee": entity.CountsTowardGuarantee,
				"defaultAmountMinor":    int64PtrValue(entity.DefaultAmountMinor),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Resolver) resolvePayProfileSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.driverPayService.GetProfile(ctx, repositories.GetPayProfileByIDRequest{
				ID:         id,
				TenantInfo: req.tenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, payProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.driverPayService.ProfileSelectOptions(
		ctx,
		&repositories.PayProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			Classification: driverpay.PayeeClassification(
				selectOptionStringFilter(req.filters, "classification"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		payProfileSelectOptionItem,
	)
}

func payProfileSelectOptionItem(entity *driverpay.PayProfile) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringPtr(entity.Description),
			Meta: map[string]any{
				"classification": string(entity.Classification),
				"currencyCode":   entity.CurrencyCode,
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func int64PtrValue(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
