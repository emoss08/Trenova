package shipmentservice

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/modeprofileservice"
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
