package distanceoverrideservice

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/keelcatalog"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride/distanceoverridedb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*distanceoverride.DistanceOverride]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(
		p.DB,
		distanceoverridedb.DistanceOverridesSchema(),
	)
	locationCheck := keelcheck.ReferenceCheck(p.DB, locationdb.LocationsSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*distanceoverride.DistanceOverride]().
			WithModelName("DistanceOverride").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithCustomReferenceCheck(
				"originLocationId",
				"Origin location does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.OriginLocationID },
				locationCheck,
			).
			WithCustomReferenceCheck(
				"destinationLocationId",
				"Destination location does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.DestinationLocationID },
				locationCheck,
			).
			WithOptionalCustomReferenceCheck(
				"customerId",
				"Customer does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.CustomerID },
				keelcheck.ReferenceCheck(p.DB, keelcatalog.MustNamed("customers")),
			).
			WithCustomRule(
				validationframework.
					NewTenantedRule[*distanceoverride.DistanceOverride](
					"origin_not_equals_destination",
				).
					OnBoth().
					WithValidation(validateOriginNotEqualsDestination),
			).
			WithCustomRule(createIntermediateStopsRule(locationCheck)).
			WithCustomRule(createUniqueRouteRule(p.DB)).
			Build(),
	}
}

func validateOriginNotEqualsDestination(
	_ context.Context,
	entity *distanceoverride.DistanceOverride,
	_ *validationframework.TenantedValidationContext,
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

func createIntermediateStopsRule(
	locationCheck validationframework.CustomReferenceCheckFunc,
) validationframework.TenantedRule[*distanceoverride.DistanceOverride] {
	return validationframework.NewTenantedRule[*distanceoverride.DistanceOverride](
		"intermediate_stops_validation",
	).
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *distanceoverride.DistanceOverride,
			valCtx *validationframework.TenantedValidationContext,
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

				exists, err := locationCheck(
					ctx,
					valCtx.OrganizationID,
					valCtx.BusinessUnitID,
					stop.LocationID,
				)
				if err != nil {
					multiErr.Add(
						field,
						errortypes.ErrSystemError,
						"Failed to validate intermediate stop location",
					)
					continue
				}

				if !exists {
					multiErr.Add(
						field,
						errortypes.ErrInvalid,
						"Intermediate stop location does not exist in your organization",
					)
				}
			}

			return nil
		})
}

func createUniqueRouteRule(
	db *postgres.Connection,
) validationframework.TenantedRule[*distanceoverride.DistanceOverride] {
	return validationframework.NewTenantedRule[*distanceoverride.DistanceOverride](
		"unique_route_signature",
	).
		OnBoth().
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *distanceoverride.DistanceOverride,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.RouteSignature == "" {
				entity.RouteSignature = entity.BuildRouteSignature()
			}

			taken, err := routeTaken(ctx, db, entity, valCtx)
			if err != nil {
				multiErr.Add(
					"__all__",
					errortypes.ErrSystemError,
					"Failed to validate distance override route uniqueness",
				)
				return nil //nolint:nilerr // validation callbacks collect field errors and intentionally continue
			}

			if taken {
				multiErr.Add(
					"intermediateStops",
					errortypes.ErrDuplicate,
					"Distance override with this route already exists",
				)
			}

			return nil
		})
}

func routeTaken(
	ctx context.Context,
	db *postgres.Connection,
	entity *distanceoverride.DistanceOverride,
	valCtx *validationframework.TenantedValidationContext,
) (bool, error) {
	return dbtx.Read(ctx, db, func(ctx context.Context) (bool, error) {
		do := distanceoverridedb.DistanceOverrides
		q := do.Where(
			do.RouteSignature.Eq(entity.RouteSignature),
			keel.When(valCtx.IsUpdate(), do.ID.Ne(entity.ID)),
		)
		if dbscope.IsSystem(ctx) {
			q = q.AllTenants().Where(
				do.OrganizationID.Eq(valCtx.OrganizationID),
				do.BusinessUnitID.Eq(valCtx.BusinessUnitID),
			)
		}

		return q.Exists(ctx, db.Keel(ctx))
	})
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
