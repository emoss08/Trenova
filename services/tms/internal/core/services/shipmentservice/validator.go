package shipmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB                        *postgres.Connection
	AssignmentRepo            repositories.AssignmentRepository
	ControlRepo               repositories.ShipmentControlRepository
	CustomerRepo              repositories.CustomerRepository
	CommodityRepo             repositories.CommodityRepository
	HazmatSegregationRuleRepo repositories.HazmatSegregationRuleRepository
	ShipmentRepo              repositories.ShipmentRepository
}

type Validator struct {
	validator      *validationframework.TenantedValidator[*shipment.Shipment]
	assignmentRepo repositories.AssignmentRepository
}

func NewValidator(p ValidatorParams) *Validator {
	builder := newValidatorBuilder(
		p.DB,
		p.ControlRepo,
		p.CustomerRepo,
		p.CommodityRepo,
		p.HazmatSegregationRuleRepo,
		p.ShipmentRepo,
	)

	return &Validator{
		validator:      builder.Build(),
		assignmentRepo: p.AssignmentRepo,
	}
}

func newValidatorBuilder(
	db *postgres.Connection,
	controlRepo repositories.ShipmentControlRepository,
	customerRepo repositories.CustomerRepository,
	commodityRepo repositories.CommodityRepository,
	hazmatRuleRepo repositories.HazmatSegregationRuleRepository,
	shipmentRepo repositories.ShipmentRepository,
) *validationframework.TenantedValidatorBuilder[*shipment.Shipment] {
	builder := validationframework.
		NewTenantedValidatorBuilder[*shipment.Shipment]().
		WithModelName("Shipment").
		WithCustomRule(createMoveValidationRule()).
		WithCustomRule(createStopValidationRule()).
		WithCustomRule(createAdditionalChargeValidationRule(controlRepo)).
		WithCustomRule(createCommodityValidationRule()).
		WithCustomRule(createShipmentStatusCoordinationRule()).
		WithCustomRule(createHazmatSegregationRule(controlRepo, commodityRepo, hazmatRuleRepo)).
		WithCustomRule(createShipmentControlPolicyRule(controlRepo, shipmentRepo)).
		WithCustomRule(createBOLValidationRule(customerRepo))

	if db == nil {
		return builder
	}

	return builder.
		WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return db.DB() })).
		WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return db.DB() })).
		WithReferenceCheck(
			"serviceTypeId",
			"service_types",
			"Service type does not exist in your organization",
			func(s *shipment.Shipment) pulid.ID { return s.ServiceTypeID },
		).
		WithReferenceCheck(
			"shipmentTypeId",
			"shipment_types",
			"Shipment type does not exist in your organization",
			func(s *shipment.Shipment) pulid.ID { return s.ShipmentTypeID },
		).
		WithReferenceCheck(
			"customerId",
			"customers",
			"Customer does not exist in your organization",
			func(s *shipment.Shipment) pulid.ID { return s.CustomerID },
		).
		WithOptionalReferenceCheck(
			"tractorTypeId",
			"equipment_types",
			"Tractor type does not exist in your organization",
			func(s *shipment.Shipment) pulid.ID { return s.TractorTypeID },
		).
		WithOptionalReferenceCheck(
			"trailerTypeId",
			"equipment_types",
			"Trailer type does not exist in your organization",
			func(s *shipment.Shipment) pulid.ID { return s.TrailerTypeID },
		).
		WithOptionalReferenceCheck(
			"formulaTemplateId",
			"formula_templates",
			"Formula template does not exist in your organization",
			func(s *shipment.Shipment) pulid.ID { return s.FormulaTemplateID },
		)
}

func createShipmentStatusCoordinationRule() validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.
		NewTenantedRule[*shipment.Shipment]("shipment_status_coordination").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *shipment.Shipment,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.Status == shipment.StatusCompleted ||
				entity.Status == shipment.StatusReadyToInvoice {
				for moveIndex, move := range entity.Moves {
					if move == nil {
						continue
					}

					if move.Status != shipment.MoveStatusCompleted {
						multiErr.Add(
							"status",
							errortypes.ErrInvalidOperation,
							"Shipment cannot reach this status until all moves are completed",
						)
						multiErr.Add(
							fmt.Sprintf("moves[%d].status", moveIndex),
							errortypes.ErrInvalidOperation,
							"Move must be completed before the shipment can reach this status",
						)
					}
				}
			}

			for moveIndex, move := range entity.Moves {
				if move == nil {
					continue
				}

				if move.Status == shipment.MoveStatusCompleted {
					for stopIndex, stop := range move.Stops {
						if stop == nil {
							continue
						}

						if stop.Status != shipment.StopStatusCompleted {
							multiErr.Add(
								fmt.Sprintf("moves[%d].status", moveIndex),
								errortypes.ErrInvalidOperation,
								"Move cannot be completed until all stops are completed",
							)
							multiErr.Add(
								fmt.Sprintf("moves[%d].stops[%d].status", moveIndex, stopIndex),
								errortypes.ErrInvalidOperation,
								"Stop must be completed before the move can be completed",
							)
						}
					}
				}
			}

			return nil
		})
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	multiErr := v.validator.ValidateCreate(ctx, entity)
	return errortypes.MergeMultiErrors(
		multiErr,
		validateResourceActualTimeline(ctx, v.assignmentRepo, nil, entity, true),
	)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	return errortypes.MergeMultiErrors(
		v.validator.ValidateUpdate(ctx, entity),
		validateResourceActualTimeline(ctx, v.assignmentRepo, nil, entity, false),
	)
}

func (v *Validator) ValidateUpdateWithOriginal(
	ctx context.Context,
	original *shipment.Shipment,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	return errortypes.MergeMultiErrors(
		v.validator.ValidateUpdate(ctx, entity),
		validateResourceActualTimeline(ctx, v.assignmentRepo, original, entity, false),
	)
}

func createBOLValidationRule(
	customerRepo repositories.CustomerRepository,
) validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.NewTenantedRule[*shipment.Shipment]("bol_validation").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *shipment.Shipment,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			customer, err := customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
				ID: entity.CustomerID,
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				CustomerFilterOptions: repositories.CustomerFilterOptions{
					IncludeBillingProfile: true,
				},
			})
			if err != nil {
				multiErr.Add(
					"customer",
					errortypes.ErrInvalid,
					"Unable to load customer billing profile",
				)
				return nil
			}

			if customer.BillingProfile.RequireBOLNumber && entity.BOL == "" {
				multiErr.Add(
					"bol",
					errortypes.ErrInvalid,
					fmt.Sprintf("%s requires a BOL number for invoicing", customer.Code),
				)
			}

			return nil
		})
}

func createShipmentControlPolicyRule(
	controlRepo repositories.ShipmentControlRepository,
	shipmentRepo repositories.ShipmentRepository,
) validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.NewTenantedRule[*shipment.Shipment]("shipment_control_policy").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *shipment.Shipment,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			control, err := loadShipmentControlForValidation(ctx, controlRepo, entity)
			if err != nil {
				multiErr.Add(
					"shipmentControl",
					errortypes.ErrInvalid,
					"Unable to load shipment control",
				)
				return nil
			}

			if entity.Weight != nil && *entity.Weight > int64(control.MaxShipmentWeightLimit) {
				multiErr.Add(
					"weight",
					errortypes.ErrInvalid,
					fmt.Sprintf("Shipment weight cannot exceed %d", control.MaxShipmentWeightLimit),
				)
			}

			if valCtx.IsCreate() || control.AllowMoveRemovals {
				return nil
			}

			original, err := loadOriginalShipmentForValidation(ctx, shipmentRepo, entity)
			if err != nil {
				multiErr.Add(
					"id",
					errortypes.ErrInvalid,
					"Unable to load the existing shipment for validation",
				)
				return nil
			}

			if hasRemovedShipmentMove(original, entity) {
				multiErr.Add(
					"moves",
					errortypes.ErrInvalidOperation,
					"Your organization does not allow move removals",
				)
				return nil
			}

			return nil
		})
}

func loadShipmentControlForValidation(
	ctx context.Context,
	controlRepo repositories.ShipmentControlRepository,
	entity *shipment.Shipment,
) (*tenant.ShipmentControl, error) {
	return controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func loadOriginalShipmentForValidation(
	ctx context.Context,
	shipmentRepo repositories.ShipmentRepository,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
}

func hasRemovedShipmentMove(original, updated *shipment.Shipment) bool {
	updatedMoveIDs := make(map[pulid.ID]struct{}, len(updated.Moves))
	for _, move := range updated.Moves {
		if move == nil || move.ID.IsNil() {
			continue
		}

		updatedMoveIDs[move.ID] = struct{}{}
	}

	for _, originalMove := range original.Moves {
		if originalMove == nil || originalMove.ID.IsNil() {
			continue
		}

		if _, ok := updatedMoveIDs[originalMove.ID]; !ok {
			return true
		}
	}

	return false
}
