package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Deps) resolvePayCodeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.DriverPayService.GetPayCode(ctx, repositories.GetPayCodeByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, payCodeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.DriverPayService.PayCodeSelectOptions(
		ctx,
		&repositories.PayCodeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			Direction: driverpay.PayCodeDirection(
				selectOptionStringFilter(req.Filters, "direction"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		payCodeSelectOptionItem,
	)
}

func payCodeSelectOptionItem(entity *driverpay.PayCode) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Code,
			Description: stringutils.Ptr(entity.Name),
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

func (r *Deps) resolvePayProfileSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.DriverPayService.GetProfile(ctx, repositories.GetPayProfileByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, payProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.DriverPayService.ProfileSelectOptions(
		ctx,
		&repositories.PayProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			Classification: driverpay.PayeeClassification(
				selectOptionStringFilter(req.Filters, "classification"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		payProfileSelectOptionItem,
	)
}

func payProfileSelectOptionItem(entity *driverpay.PayProfile) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Description),
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
