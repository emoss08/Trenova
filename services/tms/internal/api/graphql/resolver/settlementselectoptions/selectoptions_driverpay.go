package settlementselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	FuelPurchaseService *fuelpurchaseservice.Service
	IftaService         *iftaservice.Service
	DriverPayService    *driverpayservice.Service
}

func (r *Sources) PayCode(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.DriverPayService.GetPayCode(ctx, driverpayports.GetPayCodeByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, payCodeSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.DriverPayService.PayCodeSelectOptions(
		ctx,
		&driverpayports.PayCodeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			Direction: paycode.PayCodeDirection(
				selectoption.StringFilter(req.Filters, "direction"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		payCodeSelectOptionItem,
	)
}

func payCodeSelectOptionItem(entity *paycode.PayCode) selectoption.Item {
	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Code,
			Description: stringutils.Ptr(entity.Name),
			Meta: map[string]any{
				selectoption.MetaKeyCode: entity.Code,
				"name":                   entity.Name,
				"direction":              string(entity.Direction),
				"taxable":                entity.Taxable,
				"countsTowardGuarantee":  entity.CountsTowardGuarantee,
				"defaultAmountMinor":     int64PtrValue(entity.DefaultAmountMinor),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Sources) PayProfile(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.DriverPayService.GetProfile(
				ctx,
				driverpayports.GetPayProfileByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, payProfileSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.DriverPayService.ProfileSelectOptions(
		ctx,
		&driverpayports.PayProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			Classification: driverpay.PayeeClassification(
				selectoption.StringFilter(req.Filters, "classification"),
			),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		payProfileSelectOptionItem,
	)
}

func payProfileSelectOptionItem(entity *driverpay.PayProfile) selectoption.Item {
	return selectoption.ItemFor(
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

func (r *Sources) FuelCard(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.FuelPurchaseService.GetCardsByIDs(
			ctx,
			&fuelpurchaseports.GetFuelCardsByIDsRequest{
				TenantInfo: req.TenantInfo,
				IDs:        req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(req.IDs, entities, fuelCardID, fuelCardSelectOptionItem)

		return selectoption.Connection(items, len(items), 0)
	}

	entities, err := r.FuelPurchaseService.ListActiveCards(
		ctx,
		&fuelpurchaseports.ListActiveFuelCardsRequest{
			TenantInfo: req.SelectQuery.TenantInfo,
			Query:      req.SelectQuery.Query,
			Limit:      req.SelectQuery.Pagination.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	items := make([]selectoption.Item, 0, len(entities))
	for _, entity := range entities {
		items = append(items, fuelCardSelectOptionItem(entity))
	}

	return selectoption.Connection(items, len(items), req.SelectQuery.Pagination.SafeOffset())
}

// resolveIFTAJurisdictionSelectOptions pages in memory because the jurisdiction
// table is a fixed reference list of roughly seventy rows shared by every
// tenant, with no organization column to scope a query by.
func (r *Sources) IFTAJurisdiction(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	jurisdictions, err := r.IftaService.ListJurisdictions(
		ctx,
		&iftaports.ListJurisdictionsRequest{
			MembersOnly: selectoption.BoolFilter(req.Filters, "membersOnly"),
		},
	)
	if err != nil {
		return nil, err
	}

	sortJurisdictions(jurisdictions)

	if len(req.IDs) > 0 {
		wanted := make(map[string]struct{}, len(req.IDs))
		for _, id := range req.IDs {
			wanted[id.String()] = struct{}{}
		}
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, jurisdiction := range jurisdictions {
			if _, ok := wanted[jurisdiction.ID.String()]; ok {
				items = append(items, iftaJurisdictionSelectOptionItem(jurisdiction))
			}
		}

		return selectoption.Connection(items, len(items), 0)
	}

	matches := matchingJurisdictions(jurisdictions, req.SelectQuery.Query)
	offset := req.SelectQuery.Pagination.SafeOffset()
	page := sliceutils.Page(matches, offset, req.SelectQuery.Pagination.SafeLimit())

	items := make([]selectoption.Item, 0, len(page))
	for _, jurisdiction := range page {
		items = append(items, iftaJurisdictionSelectOptionItem(jurisdiction))
	}

	return selectoption.Connection(items, len(matches), offset)
}

func (r *Sources) IFTAFuelType(
	_ context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			fuelType := domaintypes.IFTAFuelType(id.String())
			if !fuelType.IsValid() {
				continue
			}
			items = append(items, iftaFuelTypeSelectOptionItem(fuelType))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	matches := matchingIFTAFuelTypes(
		req.SelectQuery.Query,
		selectoption.BoolFilter(req.Filters, "countsForIfta"),
	)
	offset := req.SelectQuery.Pagination.SafeOffset()
	page := sliceutils.Page(matches, offset, req.SelectQuery.Pagination.SafeLimit())

	items := make([]selectoption.Item, 0, len(page))
	for _, fuelType := range page {
		items = append(items, iftaFuelTypeSelectOptionItem(fuelType))
	}

	return selectoption.Connection(items, len(matches), offset)
}
