//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttimeline"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazmatsegregationruleports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB                        *postgres.Connection
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
	validator      tenantedvalidation.EntityValidator[*shipment.Shipment]
	assignmentRepo assignmentports.AssignmentRepository
	coverage       RateCoverage
}

func New(p Params) *Validator {
	builder := newValidatorBuilder(validatorDeps{
		DB:                p.DB,
		ControlRepo:       p.ControlRepo,
		CustomerRepo:      p.CustomerRepo,
		CommodityRepo:     p.CommodityRepo,
		HazmatRuleRepo:    p.HazmatSegregationRuleRepo,
		ShipmentRepo:      p.ShipmentRepo,
		EquipmentTypeRepo: p.EquipmentTypeRepo,
		BillingRepo:       p.BillingRepo,
		ProfileService:    p.ModeProfileService,
		PermitService:     p.PermitService,
	})

	return &Validator{
		validator:      builder.Build(),
		assignmentRepo: p.AssignmentRepo,
		coverage:       RateCoverage{Billing: p.BillingRepo},
	}
}

type validatorDeps struct {
	DB                *postgres.Connection
	ControlRepo       tenantports.ShipmentControlRepository
	CustomerRepo      customerports.CustomerRepository
	CommodityRepo     commodityports.CommodityRepository
	HazmatRuleRepo    hazmatsegregationruleports.HazmatSegregationRuleRepository
	ShipmentRepo      shipmentports.ShipmentRepository
	EquipmentTypeRepo equipmenttypeports.EquipmentTypeRepository
	BillingRepo       tenantports.BillingControlRepository
	ProfileService    modeprofileports.ModeProfileService
	PermitService     permitports.PermitService
}

func newValidatorBuilder(
	deps validatorDeps,
) *tenantedvalidation.Builder[*shipment.Shipment] {
	db := deps.DB
	builder := tenantedvalidation.NewBuilder[*shipment.Shipment]().
		WithModelName("Shipment").
		WithCustomRule(createMoveValidationRule()).
		WithCustomRule(createStopValidationRule()).
		WithCustomRule(createAdditionalChargeValidationRule(deps.ControlRepo)).
		WithCustomRule(createCommodityValidationRule()).
		WithCustomRule(createShipmentStatusCoordinationRule()).
		WithCustomRule(createHazmatSegregationRule(
			deps.ControlRepo, deps.CommodityRepo, deps.HazmatRuleRepo,
		)).
		WithCustomRule(createCapabilityPolicyRule(
			deps.ProfileService,
			deps.ControlRepo,
			deps.ShipmentRepo,
			deps.EquipmentTypeRepo,
			deps.PermitService,
		)).
		WithCustomRule(createBOLValidationRule(deps.CustomerRepo)).
		WithCustomRule(createRateCoverageRule(RateCoverage{Billing: deps.BillingRepo}))

	if db == nil {
		return builder
	}

	checker := keelcheck.NewChecker(db, shipmentdb.ShipmentsSchema())

	return builder.
		WithUniquenessChecker(checker).
		WithReferenceChecker(checker).
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

//nolint:gocognit // existing domain workflow is intentionally branch-heavy
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

func collectAdvisories(multiErrs ...*errortypes.MultiError) []*errortypes.AdvisoryError {
	var advisories []*errortypes.AdvisoryError
	for _, multiErr := range multiErrs {
		advisories = append(advisories, multiErr.AllAdvisories()...)
	}
	return advisories
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	multiErr, _ := v.ValidateCreateWithAdvisories(ctx, entity)
	return multiErr
}

func (v *Validator) ValidateCreateWithAdvisories(
	ctx context.Context,
	entity *shipment.Shipment,
) (*errortypes.MultiError, []*errortypes.AdvisoryError) {
	entity.ApplyEntryMethodDefault(nil)
	entity.ApplyFreightTermsDefault(nil)

	base := v.validator.ValidateCreate(ctx, entity)
	timeline := shipmenttimeline.ValidateResourceActualTimeline(
		ctx,
		v.assignmentRepo,
		nil,
		entity,
		true,
	)

	return errortypes.MergeMultiErrors(base, timeline), collectAdvisories(base, timeline)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	entity.ApplyEntryMethodDefault(nil)
	entity.ApplyFreightTermsDefault(nil)

	return errortypes.MergeMultiErrors(
		v.validator.ValidateUpdate(ctx, entity),
		shipmenttimeline.ValidateResourceActualTimeline(ctx, v.assignmentRepo, nil, entity, false),
	)
}

func (v *Validator) ValidateUpdateWithOriginal(
	ctx context.Context,
	original *shipment.Shipment,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	multiErr, _ := v.ValidateUpdateWithOriginalAndAdvisories(ctx, original, entity)
	return multiErr
}

func (v *Validator) ValidateUpdateWithOriginalAndAdvisories(
	ctx context.Context,
	original *shipment.Shipment,
	entity *shipment.Shipment,
) (*errortypes.MultiError, []*errortypes.AdvisoryError) {
	entity.ApplyEntryMethodDefault(original)
	entity.ApplyFreightTermsDefault(original)

	base := v.validator.ValidateUpdate(ctx, entity)
	timeline := shipmenttimeline.ValidateResourceActualTimeline(
		ctx,
		v.assignmentRepo,
		original,
		entity,
		false,
	)
	rating := v.coverage.refuseUnpricing(ctx, original, entity)

	return errortypes.MergeMultiErrors(base, timeline, rating), collectAdvisories(base, timeline)
}

func createBOLValidationRule(
	customerRepo customerports.CustomerRepository,
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
			customer, err := customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
				ID: entity.CustomerID,
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				CustomerFilterOptions: customerports.CustomerFilterOptions{
					IncludeBillingProfile: true,
				},
			})
			if err != nil {
				multiErr.Add(
					"customer",
					errortypes.ErrInvalid,
					"Unable to load customer billing profile",
				)
				return nil //nolint:nilerr // validation callbacks collect field errors and intentionally continue
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

func loadShipmentControlForValidation(
	ctx context.Context,
	controlRepo tenantports.ShipmentControlRepository,
	entity *shipment.Shipment,
) (*shipmentcontrol.ShipmentControl, error) {
	return controlRepo.Get(ctx, tenantports.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func loadOriginalShipmentForValidation(
	ctx context.Context,
	shipmentRepo shipmentports.ShipmentRepository,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ShipmentOptions: shipmentports.ShipmentOptions{
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
