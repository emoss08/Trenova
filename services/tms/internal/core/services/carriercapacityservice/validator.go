package carriercapacityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*carriercapacity.Posting]
}

func optionalID(id *pulid.ID) pulid.ID {
	if id == nil {
		return pulid.Nil
	}
	return *id
}

func NewValidator(p ValidatorParams) *Validator {
	stateCheck := validationframework.NewUSStateReferenceCheck(p.DB)

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*carriercapacity.Posting]().
			WithModelName("CarrierCapacityPosting").
			WithReferenceChecker(validationframework.NewBunReferenceCheckerScoped(p.DB)).
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
			WithOptionalCustomReferenceCheck(
				"originStateId",
				"Origin state does not exist",
				func(p *carriercapacity.Posting) pulid.ID { return optionalID(p.OriginStateID) },
				stateCheck,
			).
			WithOptionalCustomReferenceCheck(
				"destinationStateId",
				"Destination state does not exist",
				func(p *carriercapacity.Posting) pulid.ID { return optionalID(p.DestinationStateID) },
				stateCheck,
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *carriercapacity.Posting,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *carriercapacity.Posting,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
