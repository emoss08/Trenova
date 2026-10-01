package selectoptionsresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func (r *Deps) resolveFuelIndexSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FuelIndexRepo.GetByID(ctx, &repositories.GetFuelIndexByIDRequest{
				FuelIndexID: id,
				TenantInfo:  req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, fuelIndexSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.FuelIndexRepo.SelectOptions(
		ctx,
		&repositories.FuelIndexSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		fuelIndexSelectOptionItem,
	)
}

func (r *Deps) resolveFuelSurchargeProgramSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FuelSurchargeProgramRepo.GetByID(
				ctx,
				&repositories.GetFuelSurchargeProgramByIDRequest{
					ProgramID:  id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, fuelSurchargeProgramSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.FuelSurchargeProgramRepo.SelectOptions(
		ctx,
		&repositories.FuelSurchargeProgramSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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

func (r *Deps) resolveIFTAFuelTypeSelectOptions(
	_ context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			fuelType := domaintypes.IFTAFuelType(id.String())
			if !fuelType.IsValid() {
				continue
			}
			items = append(items, iftaFuelTypeSelectOptionItem(fuelType))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	matches := matchingIFTAFuelTypes(
		req.SelectQuery.Query,
		selectOptionBoolFilter(req.Filters, "countsForIfta"),
	)
	offset := req.SelectQuery.Pagination.SafeOffset()
	page := sliceutils.Page(matches, offset, req.SelectQuery.Pagination.SafeLimit())

	items := make([]selectOptionConnectionItem, 0, len(page))
	for _, fuelType := range page {
		items = append(items, iftaFuelTypeSelectOptionItem(fuelType))
	}

	return selectOptionConnection(items, len(matches), offset)
}

func matchingIFTAFuelTypes(query string, iftaOnly bool) []domaintypes.IFTAFuelType {
	needle := strings.ToLower(strings.TrimSpace(query))
	fuelTypes := domaintypes.AllIFTAFuelTypes()

	matches := make([]domaintypes.IFTAFuelType, 0, len(fuelTypes))
	for _, fuelType := range fuelTypes {
		if iftaOnly && !fuelType.CountsForIFTA() {
			continue
		}
		if needle != "" && !iftaFuelTypeMatches(fuelType, needle) {
			continue
		}
		matches = append(matches, fuelType)
	}

	return matches
}

func iftaFuelTypeMatches(fuelType domaintypes.IFTAFuelType, needle string) bool {
	return strings.Contains(strings.ToLower(fuelType.String()), needle) ||
		strings.Contains(strings.ToLower(fuelType.Label()), needle)
}

func iftaFuelTypeSelectOptionItem(
	fuelType domaintypes.IFTAFuelType,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		iftaFuelTypeSelectOption(fuelType),
		0,
		pulid.ID(fuelType.String()),
	)
}

func iftaFuelTypeSelectOption(fuelType domaintypes.IFTAFuelType) *gqlmodel.SelectOption {
	description := iftaFuelTypeDescription(fuelType)
	return &gqlmodel.SelectOption{
		ID:          fuelType.String(),
		Label:       fuelType.Label(),
		Description: &description,
		Meta: map[string]any{
			"countsForIfta": fuelType.CountsForIFTA(),
			"gaseous":       fuelType.IsGaseous(),
		},
	}
}

func iftaFuelTypeDescription(fuelType domaintypes.IFTAFuelType) string {
	switch {
	case !fuelType.CountsForIFTA():
		return "Not reported on the quarterly IFTA return"
	case fuelType.IsGaseous():
		return "Reported on the quarterly IFTA return in gallon equivalents"
	default:
		return "Reported on the quarterly IFTA return"
	}
}
