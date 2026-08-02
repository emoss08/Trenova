package shipmentservice

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/modeprofileservice"
	"github.com/emoss08/trenova/internal/core/services/permitservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func resolveProfileForShipment(
	ctx context.Context,
	profileSvc services.ModeProfileService,
	entity *shipment.Shipment,
) *modeprofile.ResolvedPolicy {
	if profileSvc == nil {
		return nil
	}

	policy, err := profileSvc.Resolve(ctx, &services.ResolveModeProfileRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		CustomerID:     entity.CustomerID,
		ServiceTypeID:  entity.ServiceTypeID,
		ShipmentTypeID: entity.ShipmentTypeID,
		TractorTypeID:  entity.TractorTypeID,
		TrailerTypeID:  entity.TrailerTypeID,
	})
	if err != nil {
		return nil
	}

	return policy
}

func (s *service) recordCapabilityDeviations(
	ctx context.Context,
	entity *shipment.Shipment,
	advisories []*errortypes.Advisory,
) {
	if s.modeProfileService == nil || entity == nil {
		return
	}

	policy := resolveProfileForShipment(ctx, s.modeProfileService, entity)
	if policy == nil {
		return
	}

	if err := s.modeProfileService.RecordDeviations(ctx, &services.RecordDeviationsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceType: modeprofileservice.ResourceTypeShipment,
		ResourceID:   entity.ID,
		Policy:       policy,
		Advisories:   advisories,
	}); err != nil {
		s.l.Error("failed to record capability deviations", zap.Error(err))
	}
}

func emit(
	multiErr *errortypes.MultiError,
	rule modeprofile.ResolvedRule,
	field string,
	code errortypes.ErrorCode,
	message string,
) {
	switch rule.Enforcement {
	case tenant.EnforcementLevelBlock:
		multiErr.Add(field, code, message)
	case tenant.EnforcementLevelWarn:
		multiErr.Advise(field, code, message, errortypes.SeverityWarn, rule.Key.String())
	case tenant.EnforcementLevelRequireReview:
		multiErr.Advise(field, code, message, errortypes.SeverityRequireReview, rule.Key.String())
	case tenant.EnforcementLevelIgnore:
	}
}

func maxWeightFor(
	policy *modeprofile.ResolvedPolicy,
	control *tenant.ShipmentControl,
) (int64, modeprofile.ResolvedRule, bool) {
	rule, ok := policy.Rule(modeprofile.RuleKeyMaxShipmentWeight)
	if !ok || !rule.Enabled {
		return int64(control.MaxShipmentWeightLimit), fallbackRule(
			modeprofile.RuleKeyMaxShipmentWeight,
		), true
	}

	limit, hasLimit := policy.IntParam(
		modeprofile.RuleKeyMaxShipmentWeight,
		modeprofile.ParamMaxWeight,
	)
	if !hasLimit || limit <= 0 {
		limit = int64(control.MaxShipmentWeightLimit)
	}

	return limit, rule, rule.Applies()
}

func fallbackRule(key modeprofile.RuleKey) modeprofile.ResolvedRule {
	def, err := modeprofile.RuleDefinitionFor(key)
	if err != nil {
		return modeprofile.ResolvedRule{
			Key:         key,
			Enforcement: tenant.EnforcementLevelBlock,
			Enabled:     true,
		}
	}

	return modeprofile.ResolvedRule{
		Key:         key,
		Capability:  def.Capability,
		Label:       def.Label,
		Enforcement: def.DefaultEnforcement,
		Enabled:     true,
		Fields:      def.Fields,
	}
}

func moveRemovalRuleFor(
	policy *modeprofile.ResolvedPolicy,
	control *tenant.ShipmentControl,
) modeprofile.ResolvedRule {
	if rule, ok := policy.Rule(modeprofile.RuleKeyMoveRemoval); ok {
		return rule
	}

	rule := fallbackRule(modeprofile.RuleKeyMoveRemoval)
	if control.AllowMoveRemovals {
		rule.Enforcement = tenant.EnforcementLevelIgnore
	}

	return rule
}

func createCapabilityPolicyRule(
	profileSvc services.ModeProfileService,
	controlRepo repositories.ShipmentControlRepository,
	shipmentRepo repositories.ShipmentRepository,
	equipmentTypeRepo repositories.EquipmentTypeRepository,
	permitSvc services.PermitService,
) validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.NewTenantedRule[*shipment.Shipment]("capability_policy").
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
				return nil //nolint:nilerr // validation callbacks collect field errors and intentionally continue
			}

			policy := resolveProfileForShipment(ctx, profileSvc, entity)

			validateWeight(entity, policy, control, multiErr)
			validateTemperature(entity, policy, multiErr)
			validateDimensions(entity, policy, multiErr)
			validateDeckFit(ctx, entity, policy, equipmentTypeRepo, multiErr)
			validatePermits(
				entity, policy,
				permitAssessmentFor(ctx, permitSvc, policy, entity),
				multiErr,
			)

			return validateMoveRemoval(
				ctx, entity, valCtx, policy, control, shipmentRepo, multiErr,
			)
		})
}

func validateWeight(
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	control *tenant.ShipmentControl,
	multiErr *errortypes.MultiError,
) {
	if entity.Weight == nil {
		return
	}

	limit, rule, applies := maxWeightFor(policy, control)
	if !applies || *entity.Weight <= limit {
		return
	}

	emit(multiErr, rule, "weight", errortypes.ErrInvalid,
		fmt.Sprintf("Shipment weight cannot exceed %d", limit))
}

func validateTemperature(
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	multiErr *errortypes.MultiError,
) {
	rule, ok := policy.Rule(modeprofile.RuleKeyTemperatureRange)
	if !ok || !rule.Applies() {
		return
	}

	requireBoth, _ := policy.BoolParam(
		modeprofile.RuleKeyTemperatureRange,
		modeprofile.ParamRequireBothBounds,
	)

	if entity.TemperatureMin == nil && (requireBoth || entity.TemperatureMax == nil) {
		emit(multiErr, rule, "temperatureMin", errortypes.ErrRequired,
			"Temperature controlled freight requires a minimum temperature")
	}

	if requireBoth && entity.TemperatureMax == nil {
		emit(multiErr, rule, "temperatureMax", errortypes.ErrRequired,
			"Temperature controlled freight requires a maximum temperature")
	}

	if entity.TemperatureMin != nil && entity.TemperatureMax != nil &&
		*entity.TemperatureMax < *entity.TemperatureMin {
		emit(multiErr, rule, "temperatureMax", errortypes.ErrInvalid,
			"Maximum temperature cannot be below the minimum temperature")
	}
}

func validateMoveRemoval(
	ctx context.Context,
	entity *shipment.Shipment,
	valCtx *validationframework.TenantedValidationContext,
	policy *modeprofile.ResolvedPolicy,
	control *tenant.ShipmentControl,
	shipmentRepo repositories.ShipmentRepository,
	multiErr *errortypes.MultiError,
) error {
	if valCtx.IsCreate() {
		return nil
	}

	rule := moveRemovalRuleFor(policy, control)
	if !rule.Applies() {
		return nil
	}

	original, err := loadOriginalShipmentForValidation(ctx, shipmentRepo, entity)
	if err != nil {
		multiErr.Add(
			"id",
			errortypes.ErrInvalid,
			"Unable to load the existing shipment for validation",
		)
		return nil //nolint:nilerr // validation callbacks collect field errors and intentionally continue
	}

	if !hasRemovedShipmentMove(original, entity) {
		return nil
	}

	emit(multiErr, rule, "moves", errortypes.ErrInvalidOperation,
		"Your organization does not allow move removals")

	return nil
}

func validateDimensions(
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	multiErr *errortypes.MultiError,
) {
	rule, ok := policy.Rule(modeprofile.RuleKeyDimensionsRequired)
	if !ok || !rule.Applies() {
		return
	}

	for idx, line := range entity.Commodities {
		if line == nil || line.HasDimensions() {
			continue
		}

		emit(multiErr.WithIndex("commodities", idx), rule, "", errortypes.ErrRequired,
			"Length, width, and height are required for open deck freight")
	}
}

func deckLengthFeet(
	ctx context.Context,
	entity *shipment.Shipment,
	equipmentTypeRepo repositories.EquipmentTypeRepository,
) float64 {
	if entity.TrailerTypeID.IsNil() || equipmentTypeRepo == nil {
		return 0
	}

	equipmentType, err := equipmentTypeRepo.GetByID(ctx, repositories.GetEquipmentTypeByIDRequest{
		ID: entity.TrailerTypeID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil || equipmentType.InteriorLength == nil {
		return 0
	}

	return *equipmentType.InteriorLength
}

func validateDeckFit(
	ctx context.Context,
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	equipmentTypeRepo repositories.EquipmentTypeRepository,
	multiErr *errortypes.MultiError,
) {
	rule, ok := policy.Rule(modeprofile.RuleKeyEnvelopeExceedsEquipment)
	if !ok || !rule.Applies() {
		return
	}

	envelope := entity.CurrentEnvelope()
	if envelope.LengthFeet <= 0 {
		return
	}

	deckLength := deckLengthFeet(ctx, entity, equipmentTypeRepo)
	if deckLength <= 0 {
		return
	}

	allowedOverhang, _ := policy.IntParam(
		modeprofile.RuleKeyEnvelopeExceedsEquipment,
		modeprofile.ParamMaxOverhangFeet,
	)

	overhang := envelope.LengthFeet - deckLength
	if overhang <= float64(allowedOverhang) {
		return
	}

	emit(multiErr, rule, "commodities", errortypes.ErrInvalid,
		fmt.Sprintf(
			"Cargo overhangs the deck by %.1f ft, more than the %d ft allowed",
			overhang,
			allowedOverhang,
		))
}

func permitAssessmentFor(
	ctx context.Context,
	permitSvc services.PermitService,
	policy *modeprofile.ResolvedPolicy,
	entity *shipment.Shipment,
) *services.PermitAssessment {
	if permitSvc == nil || policy == nil {
		return nil
	}

	if !policy.HasCapability(modeprofile.CapabilityPermitting) {
		return nil
	}

	assessment, err := permitSvc.Assess(ctx, entity)
	if err != nil {
		return nil
	}

	return assessment
}

func jurisdictionList(requirements []*permit.Requirement) string {
	codes := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		if code := requirement.StateCode(); code != "" {
			codes = append(codes, code)
		}
	}
	return strings.Join(codes, ", ")
}

func validatePermits(
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	assessment *services.PermitAssessment,
	multiErr *errortypes.MultiError,
) {
	if assessment == nil {
		return
	}

	validatePermitRequired(policy, assessment, multiErr)
	validatePermitExpiry(policy, assessment, multiErr)
	validatePermitEscorts(policy, assessment, multiErr)
	validatePermitLeadTime(entity, policy, assessment, multiErr)
	validatePermitCurfew(policy, assessment, multiErr)
}

func validatePermitRequired(
	policy *modeprofile.ResolvedPolicy,
	assessment *services.PermitAssessment,
	multiErr *errortypes.MultiError,
) {
	rule, ok := policy.Rule(modeprofile.RuleKeyPermitRequired)
	if !ok || !rule.Applies() || !assessment.HasOpen() {
		return
	}

	emit(multiErr, rule, "permits", errortypes.ErrRequired,
		fmt.Sprintf("Permits are required and not recorded for %s",
			jurisdictionList(assessment.Open)))
}

func validatePermitExpiry(
	policy *modeprofile.ResolvedPolicy,
	assessment *services.PermitAssessment,
	multiErr *errortypes.MultiError,
) {
	rule, ok := policy.Rule(modeprofile.RuleKeyPermitExpiry)
	if !ok || !rule.Applies() || len(assessment.ExpiringBeforeDelivery) == 0 {
		return
	}

	emit(multiErr, rule, "permits", errortypes.ErrInvalid,
		fmt.Sprintf("Permits for %s expire before the final scheduled arrival",
			jurisdictionList(assessment.ExpiringBeforeDelivery)))
}

func validatePermitEscorts(
	policy *modeprofile.ResolvedPolicy,
	assessment *services.PermitAssessment,
	multiErr *errortypes.MultiError,
) {
	rule, ok := policy.Rule(modeprofile.RuleKeyPermitEscorts)
	if !ok || !rule.Applies() || !assessment.RequiresEscorts() {
		return
	}

	emit(multiErr, rule, "permits", errortypes.ErrInvalidOperation,
		fmt.Sprintf("This route requires %d escort vehicle(s); confirm they are booked",
			assessment.TotalEscorts))
}

func validatePermitLeadTime(
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	assessment *services.PermitAssessment,
	multiErr *errortypes.MultiError,
) {
	rule, ok := policy.Rule(modeprofile.RuleKeyPermitLeadTime)
	if !ok || !rule.Applies() || assessment.MaxLeadTimeDays == 0 {
		return
	}

	pickup := permitservice.FirstPickupAt(entity)
	if pickup == 0 || pickup >= assessment.EarliestPickup {
		return
	}

	emit(multiErr, rule, "moves", errortypes.ErrInvalid,
		fmt.Sprintf(
			"Pickup is scheduled sooner than the %d day permit lead time on this route",
			assessment.MaxLeadTimeDays))
}

// validatePermitCurfew reports that movement restrictions apply rather than
// asserting a computed conflict. The seeded reference data records which
// restrictions a jurisdiction imposes but not the exact windows, and claiming a
// precise conflict from a boolean would be worse than saying nothing.
func validatePermitCurfew(
	policy *modeprofile.ResolvedPolicy,
	assessment *services.PermitAssessment,
	multiErr *errortypes.MultiError,
) {
	rule, ok := policy.Rule(modeprofile.RuleKeyPermitCurfewConflict)
	if !ok || !rule.Applies() {
		return
	}

	restricted := make([]*permit.Requirement, 0, len(assessment.Requirements))
	for _, requirement := range assessment.Requirements {
		if len(requirement.Restrictions) > 0 {
			restricted = append(restricted, requirement)
		}
	}

	if len(restricted) == 0 {
		return
	}

	emit(multiErr, rule, "moves", errortypes.ErrInvalid,
		fmt.Sprintf(
			"Movement restrictions apply in %s; confirm the appointment windows are permitted",
			jurisdictionList(restricted)))
}
