package carriercapacityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*carriercapacity.Posting]
}

func optionalID(id *pulid.ID) pulid.ID {
	if id == nil {
		return pulid.Nil
	}
	return *id
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*carriercapacity.Posting]().
			WithModelName("CarrierCapacityPosting").
			WithIntegrityChecker(p.Integrity).
			WithReferenceCheck(
				"carrierId",
				"carriers",
				"Carrier does not exist in your organization",
				func(p *carriercapacity.Posting) pulid.ID { return p.CarrierID },
			).
			WithOptionalReferenceCheck(
				"originLocationId",
				"locations",
				"Origin location does not exist in your organization",
				func(p *carriercapacity.Posting) pulid.ID { return optionalID(p.OriginLocationID) },
			).
			WithOptionalReferenceCheck(
				"equipmentTypeId",
				"equipment_types",
				"Equipment type does not exist in your organization",
				func(p *carriercapacity.Posting) pulid.ID { return optionalID(p.EquipmentTypeID) },
			).
			WithOptionalReferenceCheck(
				"originStateId",
				"us_states",
				"Origin state does not exist",
				func(p *carriercapacity.Posting) pulid.ID { return optionalID(p.OriginStateID) },
			).
			WithOptionalReferenceCheck(
				"destinationStateId",
				"us_states",
				"Destination state does not exist",
				func(p *carriercapacity.Posting) pulid.ID { return optionalID(p.DestinationStateID) },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *carriercapacity.Posting,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *carriercapacity.Posting,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
