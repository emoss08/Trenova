//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttimeline"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentnormalize"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazmatsegregationruleports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Integrity                 validationframework.IntegrityChecker
	AssignmentRepo            assignmentports.AssignmentRepository
	ControlRepo               tenantports.ShipmentControlRepository
	CustomerRepo              customerports.CustomerRepository
	CommodityRepo             commodityports.CommodityRepository
	HazmatSegregationRuleRepo hazmatsegregationruleports.HazmatSegregationRuleRepository
	ShipmentRepo              shipmentports.ShipmentRepository
	EquipmentTypeRepo         equipmenttypeports.EquipmentTypeRepository
	BillingRepo               tenantports.BillingControlRepository
	ModeProfileService        modeprofileports.ModeProfileService
	PermitService             permitports.PermitService
}

type Validator struct {
	validator      tenantedvalidation.EntityReporter[*shipment.Shipment]
	assignmentRepo assignmentports.AssignmentRepository
	coverage       RateCoverage
}

func New(p Params) *Validator {
	coverage := RateCoverage{Billing: p.BillingRepo}
	validator := tenantedvalidation.NewBuilder[*shipment.Shipment]().
		WithModelName("Shipment").
		WithIntegrityChecker(p.Integrity).
		WithCustomRule(createMoveValidationRule()).
		WithCustomRule(createStopValidationRule()).
		WithCustomRule(createAdditionalChargeValidationRule(p.ControlRepo)).
		WithCustomRule(createCommodityValidationRule()).
		WithCustomRule(createShipmentStatusCoordinationRule()).
		WithCustomRule(createHazmatSegregationRule(
			p.ControlRepo, p.CommodityRepo, p.HazmatSegregationRuleRepo,
		)).
		WithCustomRule(createCapabilityPolicyRule(
			p.ModeProfileService,
			p.ControlRepo,
			p.ShipmentRepo,
			p.EquipmentTypeRepo,
			p.PermitService,
		)).
		WithCustomRule(createBOLValidationRule(p.CustomerRepo)).
		WithCustomRule(createRateCoverageRule(coverage)).
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
		).
		Build()

	return &Validator{
		validator:      validator,
		assignmentRepo: p.AssignmentRepo,
		coverage:       coverage,
	}
}

//nolint:gocognit // existing domain workflow is intentionally branch-heavy
func createShipmentStatusCoordinationRule() validationframework.Rule[*shipment.Shipment] {
	return validationframework.
		NewRule[*shipment.Shipment]("shipment_status_coordination").
		OnBoth().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *shipment.Shipment,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if entity.Status == shipmentvocab.StatusCompleted ||
				entity.Status == shipmentvocab.StatusReadyToInvoice {
				for moveIndex, move := range entity.Moves {
					if move == nil {
						continue
					}

					if move.Status != shipmentvocab.MoveStatusCompleted {
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

				if move.Status == shipmentvocab.MoveStatusCompleted {
					for stopIndex, stop := range move.Stops {
						if stop == nil {
							continue
						}

						if stop.Status != shipmentvocab.StopStatusCompleted {
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

func (v *Validator) ValidateCreateWithAdvisories(
	ctx context.Context,
	entity *shipment.Shipment,
) ([]*errortypes.AdvisoryError, error) {
	shipmentnormalize.EntryMethod(entity, nil)
	shipmentnormalize.FreightTerms(entity, nil)

	report := v.validator.ReportCreate(ctx, entity)
	if report.Internal() != nil {
		return nil, report.Err()
	}

	merged := errortypes.MergeMultiErrors(
		report,
		validateMoveChronology(nil, entity),
		shipmenttimeline.ValidateResourceActualTimeline(ctx, v.assignmentRepo, nil, entity, true),
	)

	return merged.AllAdvisories(), merged.Err()
}

func (v *Validator) ValidateUpdateWithOriginal(
	ctx context.Context,
	original *shipment.Shipment,
	entity *shipment.Shipment,
) error {
	_, err := v.ValidateUpdateWithOriginalAndAdvisories(ctx, original, entity)
	return err
}

func (v *Validator) ValidateUpdateWithOriginalAndAdvisories(
	ctx context.Context,
	original *shipment.Shipment,
	entity *shipment.Shipment,
) ([]*errortypes.AdvisoryError, error) {
	shipmentnormalize.EntryMethod(entity, original)
	shipmentnormalize.FreightTerms(entity, original)

	report := v.validator.ReportUpdateWithOriginal(ctx, entity, original)
	if report.Internal() != nil {
		return nil, report.Err()
	}

	merged := errortypes.MergeMultiErrors(
		report,
		validateMoveChronology(original, entity),
		shipmenttimeline.ValidateResourceActualTimeline(
			ctx,
			v.assignmentRepo,
			original,
			entity,
			false,
		),
		v.coverage.refuseUnpricing(ctx, original, entity),
	)

	return merged.AllAdvisories(), merged.Err()
}

func createBOLValidationRule(
	customerRepo customerports.CustomerRepository,
) validationframework.Rule[*shipment.Shipment] {
	return validationframework.NewRule[*shipment.Shipment]("bol_validation").
		OnBoth().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			ctx context.Context,
			entity *shipment.Shipment,
			valCtx *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if multiErr.HasErrorAt("customerId") {
				return nil
			}

			customer, err := customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
				ID: entity.CustomerID,
				TenantInfo: pagination.TenantInfo{
					OrgID: valCtx.OrganizationID,
					BuID:  valCtx.BusinessUnitID,
				},
				CustomerFilterOptions: customerports.CustomerFilterOptions{
					IncludeBillingProfile: true,
				},
			})
			if err != nil {
				return err
			}

			if customer.BillingProfile.RequireBOLNumber && entity.BOL == "" {
				multiErr.Add(
					"bol",
					errortypes.ErrInvalid,
					"{0} requires a BOL number for invoicing", customer.Code,
				)
			}

			return nil
		})
}

var (
	shipmentControlKey = validationframework.NewMemoKey[*shipmentcontrol.ShipmentControl](
		"shipment_control",
	)
	originalShipmentKey = validationframework.NewMemoKey[*shipment.Shipment]("original_shipment")
)

func loadShipmentControlForValidation(
	ctx context.Context,
	controlRepo tenantports.ShipmentControlRepository,
	valCtx *validationframework.Context,
) (*shipmentcontrol.ShipmentControl, error) {
	return validationframework.Remember(
		ctx,
		valCtx,
		shipmentControlKey,
		func(ctx context.Context) (*shipmentcontrol.ShipmentControl, error) {
			return controlRepo.Get(ctx, tenantports.GetShipmentControlRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: valCtx.OrganizationID,
					BuID:  valCtx.BusinessUnitID,
				},
			})
		},
	)
}

func loadOriginalShipmentForValidation(
	ctx context.Context,
	shipmentRepo shipmentports.ShipmentRepository,
	entity *shipment.Shipment,
	valCtx *validationframework.Context,
) (*shipment.Shipment, error) {
	if original, ok := validationframework.OriginalOf[*shipment.Shipment](valCtx); ok &&
		original != nil {
		return original, nil
	}

	return validationframework.Remember(
		ctx,
		valCtx,
		originalShipmentKey,
		func(ctx context.Context) (*shipment.Shipment, error) {
			return shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
				ID: entity.ID,
				TenantInfo: pagination.TenantInfo{
					OrgID: valCtx.OrganizationID,
					BuID:  valCtx.BusinessUnitID,
				},
				ShipmentOptions: shipmentports.ShipmentOptions{
					ExpandShipmentDetails: true,
				},
			})
		},
	)
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
