package base

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func fuelSurchargeDetailToMap(detail *shipment.FuelSurchargeDetail) map[string]any {
	if detail == nil {
		return nil
	}

	data, err := sonic.Marshal(detail)
	if err != nil {
		return nil
	}

	var result map[string]any
	if err = sonic.Unmarshal(data, &result); err != nil {
		return nil
	}

	return result
}

func FuelIndexToModel(entity *fuelsurcharge.FuelIndex) *gqlmodel.FuelIndex {
	if entity == nil {
		return nil
	}

	return &gqlmodel.FuelIndex{
		ID:             entity.ID.String(),
		BusinessUnitID: entity.BusinessUnitID.String(),
		OrganizationID: entity.OrganizationID.String(),
		Name:           entity.Name,
		Code:           entity.Code,
		Description:    entity.Description,
		Source:         entity.Source,
		FuelType:       entity.FuelType,
		Region:         entity.Region,
		EiaSeriesID:    entity.EIASeriesID,
		Currency:       entity.Currency,
		IsActive:       entity.IsActive,
		Version:        int(entity.Version),
		CreatedAt:      int(entity.CreatedAt),
		UpdatedAt:      int(entity.UpdatedAt),
	}
}

func NullDecimalToStringPtr(value decimal.NullDecimal) *string {
	if !value.Valid {
		return nil
	}
	s := value.Decimal.String()
	return &s
}

func DecimalPtrToStringPtr(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	s := value.String()
	return &s
}

func NullDecimalFromStringPtr(value *string, field string) (decimal.NullDecimal, error) {
	if value == nil || *value == "" {
		return decimal.NullDecimal{}, nil
	}

	parsed, err := decimal.NewFromString(*value)
	if err != nil {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field, errortypes.ErrInvalid, "Must be a valid decimal number")
	}

	return decimal.NewNullDecimal(parsed), nil
}

func DecimalFromString(value, field string) (decimal.Decimal, error) {
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Decimal{}, errortypes.NewValidationError(
			field, errortypes.ErrInvalid, "Must be a valid decimal number")
	}

	return parsed, nil
}

func PulidsFromStrings(values []string, field string) ([]pulid.ID, error) {
	if len(values) == 0 {
		return nil, nil
	}

	return ParsePulids(field, values)
}

func Int64PtrToIntPtr(value *int64) *int {
	if value == nil {
		return nil
	}
	v := int(*value)
	return &v
}
