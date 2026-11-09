package distanceoverrideservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	validator tenantedvalidation.EntityValidator[*distanceoverride.DistanceOverride]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*distanceoverride.DistanceOverride]().
			WithModelName("DistanceOverride").
			WithIntegrityChecker(p.Integrity).
			WithReferenceCheck(
				"originLocationId",
				"locations",
				"Origin location does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.OriginLocationID },
			).
			WithReferenceCheck(
				"destinationLocationId",
				"locations",
				"Destination location does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.DestinationLocationID },
			).
			WithOptionalReferenceCheck(
				"customerId",
				"customers",
				"Customer does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.CustomerID },
			).
			WithCustomRule(
				validationframework.
					NewRule[*distanceoverride.DistanceOverride](
					"origin_not_equals_destination",
				).
					OnBoth().
					WithValidation(validateOriginNotEqualsDestination),
			).
			WithCustomRule(createIntermediateStopsRule()).
			WithReferenceList(tenantedvalidation.ReferenceListConfig[*distanceoverride.DistanceOverride]{
				Table:   "locations",
				Message: "Intermediate stop location does not exist in your organization",
				Targets: intermediateStopTargets,
			}).
			WithCaseSensitiveUniqueField(
				"intermediateStops",
				"route_signature",
				"Distance override with this route already exists",
				func(d *distanceoverride.DistanceOverride) any { return d.RouteSignature },
			).
			Build(),
	}
}

func validateOriginNotEqualsDestination(
	_ context.Context,
	entity *distanceoverride.DistanceOverride,
	_ *validationframework.Context,
	multiErr *errortypes.MultiError,
) error {
	if !entity.OriginLocationID.IsNil() && !entity.DestinationLocationID.IsNil() &&
		entity.OriginLocationID == entity.DestinationLocationID {
		multiErr.Add(
			"destinationLocationId",
			errortypes.ErrInvalid,
			"Destination location must be different from origin location",
		)
	}
	return nil
}

func createIntermediateStopsRule() validationframework.Rule[*distanceoverride.DistanceOverride] {
	return validationframework.NewRule[*distanceoverride.DistanceOverride](
		"intermediate_stops_validation",
	).
		OnBoth().
		WithStage(validationframework.StageBasic).
		WithValidation(func(
			_ context.Context,
			entity *distanceoverride.DistanceOverride,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			seen := make(map[pulid.ID]struct{}, len(entity.IntermediateStops))

			for idx, stop := range entity.IntermediateStops {
				field := fmt.Sprintf("intermediateStops[%d].locationId", idx)
				if stop == nil || stop.LocationID.IsNil() {
					multiErr.Add(
						field,
						errortypes.ErrRequired,
						"Intermediate stop location is required",
					)
					continue
				}

				if stop.LocationID == entity.OriginLocationID {
					multiErr.Add(
						field,
						errortypes.ErrInvalid,
						"Intermediate stop cannot match origin location",
					)
				}

				if stop.LocationID == entity.DestinationLocationID {
					multiErr.Add(
						field,
						errortypes.ErrInvalid,
						"Intermediate stop cannot match destination location",
					)
				}

				if _, ok := seen[stop.LocationID]; ok {
					multiErr.Add(
						field,
						errortypes.ErrDuplicate,
						"Intermediate stop location must be unique within the route",
					)
					continue
				}

				seen[stop.LocationID] = struct{}{}
			}

			return nil
		})
}

func intermediateStopTargets(
	entity *distanceoverride.DistanceOverride,
) []tenantedvalidation.ReferenceTarget {
	targets := make([]tenantedvalidation.ReferenceTarget, 0, len(entity.IntermediateStops))
	for idx, stop := range entity.IntermediateStops {
		if stop == nil || stop.LocationID.IsNil() {
			continue
		}
		targets = append(targets, tenantedvalidation.ReferenceTarget{
			FieldName: fmt.Sprintf("intermediateStops[%d].locationId", idx),
			ID:        stop.LocationID,
		})
	}

	return targets
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
