package shipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubModeProfileService struct {
	policy *modeprofile.ResolvedPolicy
}

func (s *stubModeProfileService) Resolve(
	_ context.Context,
	_ *services.ResolveModeProfileRequest,
) (*modeprofile.ResolvedPolicy, error) {
	return s.policy, nil
}

func (s *stubModeProfileService) RecordDeviations(
	_ context.Context,
	_ *services.RecordDeviationsRequest,
) error {
	return nil
}

func (s *stubModeProfileService) Acknowledge(
	_ context.Context,
	_ *services.AcknowledgeDeviationServiceRequest,
) (*modeprofile.Deviation, error) {
	return nil, nil
}

func (s *stubModeProfileService) ListDeviations(
	_ context.Context,
	_ *repositories.ListDeviationsRequest,
) (*pagination.ListResult[*modeprofile.Deviation], error) {
	return nil, nil
}

func (s *stubModeProfileService) Ledger(
	_ context.Context,
	_ *repositories.DeviationLedgerRequest,
) ([]*repositories.DeviationLedgerEntry, error) {
	return nil, nil
}

func policyWithWeightRule(
	enforcement tenant.EnforcementLevel,
	maxWeight int64,
) *modeprofile.ResolvedPolicy {
	return &modeprofile.ResolvedPolicy{
		ProfileCode:  "OTR",
		Capabilities: []modeprofile.Capability{modeprofile.CapabilityCore},
		Rules: map[modeprofile.RuleKey]modeprofile.ResolvedRule{
			modeprofile.RuleKeyMaxShipmentWeight: {
				Key:         modeprofile.RuleKeyMaxShipmentWeight,
				Capability:  modeprofile.CapabilityCore,
				Enforcement: enforcement,
				Enabled:     true,
				Parameters:  map[string]any{modeprofile.ParamMaxWeight: maxWeight},
			},
		},
	}
}

func validatorWithProfile(
	t *testing.T,
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	control *tenant.ShipmentControl,
) *Validator {
	t.Helper()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(control, nil).
		Maybe()

	return &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			mocks.NewMockCommodityRepository(t),
			mocks.NewMockHazmatSegregationRuleRepository(t),
			mocks.NewMockShipmentRepository(t),
			&stubModeProfileService{policy: policy},
		).Build(),
	}
}

func TestCapabilityPolicy_BlockEnforcementFailsValidation(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	weight := int64(90000)
	entity.Weight = &weight

	v := validatorWithProfile(
		t,
		entity,
		policyWithWeightRule(tenant.EnforcementLevelBlock, 80000),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.NotNil(t, multiErr)
	require.True(t, multiErr.HasErrors())
	require.Empty(t, advisories)

	requireFieldError(t, multiErr, "weight", "Shipment weight cannot exceed 80000")
}

func TestCapabilityPolicy_WarnEnforcementPassesWithAdvisory(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	weight := int64(90000)
	entity.Weight = &weight

	v := validatorWithProfile(
		t,
		entity,
		policyWithWeightRule(tenant.EnforcementLevelWarn, 80000),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Len(t, advisories, 1)

	advisory := advisories[0]
	require.Equal(t, "weight", advisory.Field)
	require.Equal(t, errortypes.SeverityWarn, advisory.Severity)
	require.Equal(t, modeprofile.RuleKeyMaxShipmentWeight.String(), advisory.RuleKey)
}

func TestCapabilityPolicy_RequireReviewProducesReviewAdvisory(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	weight := int64(90000)
	entity.Weight = &weight

	v := validatorWithProfile(
		t,
		entity,
		policyWithWeightRule(tenant.EnforcementLevelRequireReview, 80000),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Len(t, advisories, 1)
	require.Equal(t, errortypes.SeverityRequireReview, advisories[0].Severity)
}

func TestCapabilityPolicy_IgnoreEnforcementSkipsRuleEntirely(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	weight := int64(90000)
	entity.Weight = &weight

	v := validatorWithProfile(
		t,
		entity,
		policyWithWeightRule(tenant.EnforcementLevelIgnore, 80000),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

func TestCapabilityPolicy_ProfileParameterOverridesShipmentControlLimit(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	weight := int64(90000)
	entity.Weight = &weight

	v := validatorWithProfile(
		t,
		entity,
		policyWithWeightRule(tenant.EnforcementLevelBlock, 100000),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

func TestCapabilityPolicy_TemperatureRuleRequiresRangeWhenCapabilityDeclared(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.TemperatureMin = nil
	entity.TemperatureMax = nil

	policy := &modeprofile.ResolvedPolicy{
		ProfileCode: "REEFER",
		Capabilities: []modeprofile.Capability{
			modeprofile.CapabilityCore,
			modeprofile.CapabilityTemperatureControl,
		},
		Rules: map[modeprofile.RuleKey]modeprofile.ResolvedRule{
			modeprofile.RuleKeyTemperatureRange: {
				Key:         modeprofile.RuleKeyTemperatureRange,
				Capability:  modeprofile.CapabilityTemperatureControl,
				Enforcement: tenant.EnforcementLevelBlock,
				Enabled:     true,
				Parameters:  map[string]any{modeprofile.ParamRequireBothBounds: true},
			},
		},
	}

	v := validatorWithProfile(t, entity, policy, &tenant.ShipmentControl{
		MaxShipmentWeightLimit: 80000,
	})

	multiErr, _ := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.NotNil(t, multiErr)
	requireFieldError(
		t, multiErr, "temperatureMin",
		"Temperature controlled freight requires a minimum temperature",
	)
	requireFieldError(
		t, multiErr, "temperatureMax",
		"Temperature controlled freight requires a maximum temperature",
	)
}

func TestCapabilityPolicy_NoProfileFallsBackToShipmentControl(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	weight := int64(90000)
	entity.Weight = &weight

	v := validatorWithProfile(t, entity, nil, &tenant.ShipmentControl{
		MaxShipmentWeightLimit: 80000,
	})

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.NotNil(t, multiErr)
	require.Empty(t, advisories)
	requireFieldError(t, multiErr, "weight", "Shipment weight cannot exceed 80000")
}

func requireFieldError(
	t *testing.T,
	multiErr *errortypes.MultiError,
	field string,
	message string,
) {
	t.Helper()

	for _, err := range multiErr.Errors {
		if err.Field == field && err.Message == message {
			return
		}
	}

	t.Fatalf("expected error on %q with message %q, got %v", field, message, multiErr.Errors)
}
