package shipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

	return validatorWithProfileAndEquipment(t, entity, policy, control, nil)
}

func validatorWithProfileAndEquipment(
	t *testing.T,
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	control *tenant.ShipmentControl,
	equipmentTypeRepo repositories.EquipmentTypeRepository,
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
			equipmentTypeRepo,
			&stubModeProfileService{policy: policy},
			nil,
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

func dimensionalPolicy(
	enforcement tenant.EnforcementLevel,
	rules ...modeprofile.RuleKey,
) *modeprofile.ResolvedPolicy {
	resolved := make(map[modeprofile.RuleKey]modeprofile.ResolvedRule, len(rules))
	for _, key := range rules {
		def, err := modeprofile.RuleDefinitionFor(key)
		if err != nil {
			continue
		}
		resolved[key] = modeprofile.ResolvedRule{
			Key:         key,
			Capability:  def.Capability,
			Label:       def.Label,
			Enforcement: enforcement,
			Enabled:     true,
			Fields:      def.Fields,
		}
	}

	return &modeprofile.ResolvedPolicy{
		ProfileCode: "FLATBED",
		Capabilities: []modeprofile.Capability{
			modeprofile.CapabilityCore,
			modeprofile.CapabilityDimensionalCargo,
		},
		Rules: resolved,
	}
}

func dimensionedLine(length, width, height float64) *shipment.ShipmentCommodity {
	return &shipment.ShipmentCommodity{
		CommodityID: pulid.MustNew("com_"),
		Weight:      1000,
		Pieces:      1,
		LengthFeet:  &length,
		WidthFeet:   &width,
		HeightFeet:  &height,
	}
}

func TestCapabilityPolicy_DimensionsRequiredBlocksLinesMissingDimensions(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Commodities = []*shipment.ShipmentCommodity{
		dimensionedLine(20, 8, 6),
		{CommodityID: pulid.MustNew("com_"), Weight: 1000, Pieces: 1},
	}

	v := validatorWithProfile(
		t,
		entity,
		dimensionalPolicy(tenant.EnforcementLevelBlock, modeprofile.RuleKeyDimensionsRequired),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, _ := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.NotNil(t, multiErr)
	requireFieldError(
		t, multiErr, "commodities[1]",
		"Length, width, and height are required for open deck freight",
	)
}

func TestCapabilityPolicy_DimensionsRequiredPassesWhenAllLinesDimensioned(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Commodities = []*shipment.ShipmentCommodity{
		dimensionedLine(20, 8, 6),
		dimensionedLine(15, 7, 5),
	}

	v := validatorWithProfile(
		t,
		entity,
		dimensionalPolicy(tenant.EnforcementLevelBlock, modeprofile.RuleKeyDimensionsRequired),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

func TestCapabilityPolicy_DimensionsRequiredWarnRecordsAdvisory(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Commodities = []*shipment.ShipmentCommodity{
		{CommodityID: pulid.MustNew("com_"), Weight: 1000, Pieces: 1},
	}

	v := validatorWithProfile(
		t,
		entity,
		dimensionalPolicy(tenant.EnforcementLevelWarn, modeprofile.RuleKeyDimensionsRequired),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Len(t, advisories, 1)
	require.Equal(t, "commodities[0]", advisories[0].Field)
	require.Equal(
		t,
		modeprofile.RuleKeyDimensionsRequired.String(),
		advisories[0].RuleKey,
	)
}

func TestCapabilityPolicy_DimensionsNotCheckedWithoutCapability(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Commodities = []*shipment.ShipmentCommodity{
		{CommodityID: pulid.MustNew("com_"), Weight: 1000, Pieces: 1},
	}

	v := validatorWithProfile(
		t,
		entity,
		policyWithWeightRule(tenant.EnforcementLevelBlock, 80000),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

func equipmentRepoWithDeck(
	t *testing.T,
	entity *shipment.Shipment,
	deckLengthFeet float64,
) repositories.EquipmentTypeRepository {
	t.Helper()

	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetEquipmentTypeByIDRequest{
			ID: entity.TrailerTypeID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&equipmenttype.EquipmentType{InteriorLength: &deckLengthFeet}, nil).
		Maybe()

	return repo
}

func TestCapabilityPolicy_DeckFitBlocksOverhangBeyondAllowance(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.TrailerTypeID = pulid.MustNew("et_")
	entity.ApplyEnvelope(shipment.Envelope{LengthFeet: 62, WidthFeet: 8, HeightFeet: 6})

	v := validatorWithProfileAndEquipment(
		t,
		entity,
		dimensionalPolicy(
			tenant.EnforcementLevelBlock,
			modeprofile.RuleKeyEnvelopeExceedsEquipment,
		),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
		equipmentRepoWithDeck(t, entity, 53),
	)

	multiErr, _ := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.NotNil(t, multiErr)
	requireFieldError(
		t, multiErr, "commodities",
		"Cargo overhangs the deck by 9.0 ft, more than the 4 ft allowed",
	)
}

func TestCapabilityPolicy_DeckFitAllowsOverhangWithinAllowance(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.TrailerTypeID = pulid.MustNew("et_")
	entity.ApplyEnvelope(shipment.Envelope{LengthFeet: 56, WidthFeet: 8, HeightFeet: 6})

	v := validatorWithProfileAndEquipment(
		t,
		entity,
		dimensionalPolicy(
			tenant.EnforcementLevelBlock,
			modeprofile.RuleKeyEnvelopeExceedsEquipment,
		),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
		equipmentRepoWithDeck(t, entity, 53),
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

func TestCapabilityPolicy_DeckFitSkippedWithoutEnvelope(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.TrailerTypeID = pulid.MustNew("et_")

	v := validatorWithProfileAndEquipment(
		t,
		entity,
		dimensionalPolicy(
			tenant.EnforcementLevelBlock,
			modeprofile.RuleKeyEnvelopeExceedsEquipment,
		),
		&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000},
		equipmentRepoWithDeck(t, entity, 53),
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

type stubPermitService struct {
	assessment *services.PermitAssessment
}

func (s *stubPermitService) Assess(
	_ context.Context, _ *shipment.Shipment,
) (*services.PermitAssessment, error) {
	return s.assessment, nil
}

func (s *stubPermitService) Sync(
	_ context.Context, _ *shipment.Shipment, _ *services.RequestActor,
) (*services.PermitAssessment, error) {
	return s.assessment, nil
}

func (s *stubPermitService) ListPermits(
	_ context.Context, _ pulid.ID, _ pagination.TenantInfo,
) ([]*permit.Permit, error) {
	return nil, nil
}

func (s *stubPermitService) ListRequirements(
	_ context.Context, _ pulid.ID, _ pagination.TenantInfo,
) ([]*permit.Requirement, error) {
	return nil, nil
}

func (s *stubPermitService) CreatePermit(
	_ context.Context, e *permit.Permit, _ *services.RequestActor,
) (*permit.Permit, error) {
	return e, nil
}

func (s *stubPermitService) UpdatePermit(
	_ context.Context, e *permit.Permit, _ *services.RequestActor,
) (*permit.Permit, error) {
	return e, nil
}

func (s *stubPermitService) WaiveRequirement(
	_ context.Context, _ *services.WaiveRequirementRequest,
) (*permit.Requirement, error) {
	return nil, nil
}

func permittingPolicy(
	enforcement tenant.EnforcementLevel,
	keys ...modeprofile.RuleKey,
) *modeprofile.ResolvedPolicy {
	rules := make(map[modeprofile.RuleKey]modeprofile.ResolvedRule, len(keys))
	for _, key := range keys {
		def, err := modeprofile.RuleDefinitionFor(key)
		if err != nil {
			continue
		}
		rules[key] = modeprofile.ResolvedRule{
			Key:         key,
			Capability:  def.Capability,
			Label:       def.Label,
			Enforcement: enforcement,
			Enabled:     true,
			Fields:      def.Fields,
		}
	}

	return &modeprofile.ResolvedPolicy{
		ProfileCode: "HEAVYHAUL",
		Capabilities: []modeprofile.Capability{
			modeprofile.CapabilityCore,
			modeprofile.CapabilityPermitting,
		},
		Rules: rules,
	}
}

func openRequirement(code string) *permit.Requirement {
	return &permit.Requirement{
		Status:     permit.RequirementOpen,
		Provenance: &permit.RuleProvenance{StateCode: code},
	}
}

func validatorWithPermits(
	t *testing.T,
	entity *shipment.Shipment,
	policy *modeprofile.ResolvedPolicy,
	assessment *services.PermitAssessment,
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
		Return(&tenant.ShipmentControl{MaxShipmentWeightLimit: 80000}, nil).
		Maybe()

	return &Validator{
		validator: newValidatorBuilder(
			nil,
			controlRepo,
			NewTestCustomerRepository(t),
			mocks.NewMockCommodityRepository(t),
			mocks.NewMockHazmatSegregationRuleRepository(t),
			mocks.NewMockShipmentRepository(t),
			nil,
			&stubModeProfileService{policy: policy},
			&stubPermitService{assessment: assessment},
		).Build(),
	}
}

func TestPermitRule_MissingPermitBlocks(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	open := []*permit.Requirement{openRequirement("GA"), openRequirement("OH")}

	v := validatorWithPermits(t, entity,
		permittingPolicy(tenant.EnforcementLevelBlock, modeprofile.RuleKeyPermitRequired),
		&services.PermitAssessment{Requirements: open, Open: open},
	)

	multiErr, _ := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.NotNil(t, multiErr)
	requireFieldError(t, multiErr, "permits",
		"Permits are required and not recorded for GA, OH")
}

func TestPermitRule_WarnRecordsAdvisoryInsteadOfBlocking(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	open := []*permit.Requirement{openRequirement("NE")}

	v := validatorWithPermits(t, entity,
		permittingPolicy(tenant.EnforcementLevelWarn, modeprofile.RuleKeyPermitRequired),
		&services.PermitAssessment{Requirements: open, Open: open},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Len(t, advisories, 1)
	require.Equal(t, modeprofile.RuleKeyPermitRequired.String(), advisories[0].RuleKey)
}

func TestPermitRule_SatisfiedRequirementsDoNotFire(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	satisfied := []*permit.Requirement{{
		Status:     permit.RequirementSatisfied,
		Provenance: &permit.RuleProvenance{StateCode: "GA"},
	}}

	v := validatorWithPermits(t, entity,
		permittingPolicy(tenant.EnforcementLevelBlock, modeprofile.RuleKeyPermitRequired),
		&services.PermitAssessment{Requirements: satisfied},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

func TestPermitRule_ExpiryBeforeDeliveryBlocks(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	expiring := []*permit.Requirement{openRequirement("UT")}

	v := validatorWithPermits(t, entity,
		permittingPolicy(tenant.EnforcementLevelBlock, modeprofile.RuleKeyPermitExpiry),
		&services.PermitAssessment{ExpiringBeforeDelivery: expiring},
	)

	multiErr, _ := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.NotNil(t, multiErr)
	requireFieldError(t, multiErr, "permits",
		"Permits for UT expire before the final scheduled arrival")
}

func TestPermitRule_EscortsDefaultToReview(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()

	v := validatorWithPermits(t, entity,
		permittingPolicy(
			tenant.EnforcementLevelRequireReview, modeprofile.RuleKeyPermitEscorts,
		),
		&services.PermitAssessment{TotalEscorts: 2},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Len(t, advisories, 1)
	require.Equal(t, errortypes.SeverityRequireReview, advisories[0].Severity)
	require.Contains(t, advisories[0].Message, "2 escort vehicle(s)")
}

func TestPermitRule_CurfewReportsRestrictionsWithoutClaimingAConflict(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	restricted := openRequirement("OH")
	restricted.Restrictions = []jurisdictionrule.RestrictionKind{
		jurisdictionrule.RestrictionRushHour,
	}

	v := validatorWithPermits(t, entity,
		permittingPolicy(
			tenant.EnforcementLevelWarn, modeprofile.RuleKeyPermitCurfewConflict,
		),
		&services.PermitAssessment{Requirements: []*permit.Requirement{restricted}},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Len(t, advisories, 1)
	require.Contains(t, advisories[0].Message, "Movement restrictions apply in OH")
	require.Contains(t, advisories[0].Message, "confirm the appointment windows")
}

// The policy carries the permit rules but does not declare the Permitting
// capability, so the assessment must never be consulted. Without this the
// capability gate could be deleted and nothing would notice.
func TestPermitRule_IgnoredWithoutPermittingCapability(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	open := []*permit.Requirement{openRequirement("GA")}

	policy := permittingPolicy(
		tenant.EnforcementLevelBlock, modeprofile.RuleKeyPermitRequired,
	)
	policy.Capabilities = []modeprofile.Capability{modeprofile.CapabilityCore}

	v := validatorWithPermits(t, entity, policy,
		&services.PermitAssessment{Requirements: open, Open: open},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

func TestPermitRule_NoEscortsRequiredProducesNothing(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()

	v := validatorWithPermits(t, entity,
		permittingPolicy(
			tenant.EnforcementLevelRequireReview, modeprofile.RuleKeyPermitEscorts,
		),
		&services.PermitAssessment{TotalEscorts: 0},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}

func TestPermitRule_LeadTimeWarnsWhenPickupIsTooSoon(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	pickup := entity.Moves[0].Stops[0].ScheduledWindowStart

	v := validatorWithPermits(t, entity,
		permittingPolicy(
			tenant.EnforcementLevelWarn, modeprofile.RuleKeyPermitLeadTime,
		),
		&services.PermitAssessment{
			MaxLeadTimeDays: 5,
			EarliestPickup:  pickup + 1,
		},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Len(t, advisories, 1)
	require.Contains(t, advisories[0].Message, "5 day permit lead time")
}

func TestPermitRule_LeadTimeSilentWhenPickupIsFarEnoughOut(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	pickup := entity.Moves[0].Stops[0].ScheduledWindowStart

	v := validatorWithPermits(t, entity,
		permittingPolicy(
			tenant.EnforcementLevelWarn, modeprofile.RuleKeyPermitLeadTime,
		),
		&services.PermitAssessment{
			MaxLeadTimeDays: 5,
			EarliestPickup:  pickup - 1,
		},
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.Nil(t, multiErr)
	require.Empty(t, advisories)
}
