package distanceoverrideservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*distanceoverride.DistanceOverride]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*distanceoverride.DistanceOverride]().
			WithModelName("DistanceOverride").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithCustomReferenceCheck(
				"originLocationId",
				"Origin location does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.OriginLocationID },
				createLocationCheck(p.DB),
			).
			WithCustomReferenceCheck(
				"destinationLocationId",
				"Destination location does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.DestinationLocationID },
				createLocationCheck(p.DB),
			).
			WithOptionalCustomReferenceCheck(
				"customerId",
				"Customer does not exist in your organization",
				func(d *distanceoverride.DistanceOverride) pulid.ID { return d.CustomerID },
				createCustomerCheck(p.DB),
			).
			WithCustomRule(
				validationframework.
					NewTenantedRule[*distanceoverride.DistanceOverride](
					"origin_not_equals_destination",
				).
					OnBoth().
					WithValidation(validateOriginNotEqualsDestination),
			).
			WithCustomRule(createIntermediateStopsRule(p.DB)).
			WithCustomRule(createUniqueRouteRule(p.DB)).
			Build(),
	}
}

func createLocationCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		exists, err := db.DB().NewSelect().
			TableExpr("locations").
			ColumnExpr("1").
			Where("id = ?", refID).
			Where("organization_id = ?", orgID).
			Where("business_unit_id = ?", buID).
			Exists(ctx)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
}

func createCustomerCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		exists, err := db.DB().NewSelect().
			TableExpr("customers").
			ColumnExpr("1").
			Where("id = ?", refID).
			Where("organization_id = ?", orgID).
			Where("business_unit_id = ?", buID).
			Exists(ctx)
		if err != nil {
			return false, err
		}
		return exists, nil
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
	db *postgres.Connection,
) validationframework.TenantedRule[*distanceoverride.DistanceOverride] {
	locationCheck := createLocationCheck(db)

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

			q := db.DB().NewSelect().
				Model((*distanceoverride.DistanceOverride)(nil)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Where("diso.organization_id = ?", valCtx.OrganizationID).
						Where("diso.business_unit_id = ?", valCtx.BusinessUnitID).
						Where("diso.route_signature = ?", entity.RouteSignature)
				})

			if valCtx.IsUpdate() {
				q = q.Where("diso.id != ?", entity.ID)
			}

			count, err := q.Count(ctx)
			if err != nil {
				multiErr.Add(
					"__all__",
					errortypes.ErrSystemError,
					"Failed to validate distance override route uniqueness",
				)
				return nil
			}

			if count > 0 {
				multiErr.Add(
					"intermediateStops",
					errortypes.ErrDuplicate,
					"Distance override with this route already exists",
				)
			}

			return nil
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
