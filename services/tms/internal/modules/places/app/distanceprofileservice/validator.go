package distanceprofileservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateCreate(
	_ context.Context,
	entity *distanceprofile.DistanceProfile,
) error {
	return v.validate(entity)
}

func (v *Validator) ValidateUpdate(
	_ context.Context,
	entity *distanceprofile.DistanceProfile,
) error {
	return v.validate(entity)
}

func (v *Validator) validate(entity *distanceprofile.DistanceProfile) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if entity.OrganizationID.IsNil() {
		multiErr.Add("organizationId", errortypes.ErrRequired, "Organization is required")
	}
	if entity.BusinessUnitID.IsNil() {
		multiErr.Add("businessUnitId", errortypes.ErrRequired, "Business unit is required")
	}
	return multiErr.Err()
}
