package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

func (r *Resolver) resolveFuelIndexSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.fuelIndexRepo.GetByID(ctx, &repositories.GetFuelIndexByIDRequest{
				FuelIndexID: id,
				TenantInfo:  req.tenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, fuelIndexSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.fuelIndexRepo.SelectOptions(
		ctx,
		&repositories.FuelIndexSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		fuelIndexSelectOptionItem,
	)
}

func (r *Resolver) resolveFuelSurchargeProgramSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.fuelSurchargeProgramRepo.GetByID(
				ctx,
				&repositories.GetFuelSurchargeProgramByIDRequest{
					ProgramID:  id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, fuelSurchargeProgramSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.fuelSurchargeProgramRepo.SelectOptions(
		ctx,
		&repositories.FuelSurchargeProgramSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		fuelSurchargeProgramSelectOptionItem,
	)
}

func fuelIndexSelectOptionItem(entity *fuelsurcharge.FuelIndex) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		fuelIndexSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func fuelIndexSelectOption(entity *fuelsurcharge.FuelIndex) *gqlmodel.SelectOption {
	description := entity.Name
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: &description,
		Meta: map[string]any{
			"source":      string(entity.Source),
			"fuelType":    string(entity.FuelType),
			"region":      entity.Region,
			"eiaSeriesId": entity.EIASeriesID,
			"currency":    entity.Currency,
		},
	}
}

func fuelSurchargeProgramSelectOptionItem(
	entity *fuelsurcharge.FuelSurchargeProgram,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		fuelSurchargeProgramSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func fuelSurchargeProgramSelectOption(
	entity *fuelsurcharge.FuelSurchargeProgram,
) *gqlmodel.SelectOption {
	description := entity.Name
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: &description,
		Meta: map[string]any{
			"method": string(entity.Method),
			"status": string(entity.Status),
		},
	}
}
