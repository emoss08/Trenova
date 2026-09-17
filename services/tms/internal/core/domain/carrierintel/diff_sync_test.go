package carrierintel_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func changeByPath(changes []carrierintel.FieldChange, path string) *carrierintel.FieldChange {
	for idx := range changes {
		if changes[idx].Path == path {
			return &changes[idx]
		}
	}
	return nil
}

func TestDiffProfiles_DetectsMeaningfulChanges(t *testing.T) {
	t.Parallel()

	prior := healthyProfile()
	current := healthyProfile()
	current.Authority.Common.Status = carrierintel.AuthorityStatusRevoked
	current.Insurance.BIPDOnFile = dec(0)
	current.Basics[0].Alert = true
	current.Authority.Common.AgeDays = new(901)

	changes, err := carrierintel.DiffProfiles(prior, current, carrierintel.DiffOptions{})
	require.NoError(t, err)

	authority := changeByPath(changes, "authority.common.status")
	require.NotNil(t, authority)
	assert.Equal(t, carrierintel.SeverityCritical, authority.Severity)
	assert.Equal(t, "Active", authority.Prior)
	assert.Equal(t, "Revoked", authority.Current)

	bipd := changeByPath(changes, "insurance.bipdOnFile")
	require.NotNil(t, bipd)
	assert.Equal(t, carrierintel.SectionInsurance, bipd.Section)

	basic := changeByPath(changes, "basics.UnsafeDriving.alert")
	require.NotNil(t, basic)
	assert.Equal(t, carrierintel.SeverityHigh, basic.Severity)

	assert.Nil(t, changeByPath(changes, "authority.common.ageDays"))
}

func TestDiffProfiles_IgnoresSectionsNotCoveredByBoth(t *testing.T) {
	t.Parallel()

	prior := healthyProfile()
	prior.Network = nil
	prior.Coverage = nil
	prior.NormalizeCoverage()

	current := healthyProfile()
	current.Network.SharedPhones = new(4)

	changes, err := carrierintel.DiffProfiles(prior, current, carrierintel.DiffOptions{})
	require.NoError(t, err)
	assert.Nil(t, changeByPath(changes, "network.sharedPhones"))
}

func TestDiffProfiles_Identical(t *testing.T) {
	t.Parallel()

	changes, err := carrierintel.DiffProfiles(
		healthyProfile(),
		healthyProfile(),
		carrierintel.DiffOptions{},
	)
	require.NoError(t, err)
	assert.Empty(t, changes)

	a, err := healthyProfile().ContentHash()
	require.NoError(t, err)
	b, err := healthyProfile().ContentHash()
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestDiffOptionsForDepths(t *testing.T) {
	t.Parallel()

	assert.False(t, carrierintel.DiffOptionsForDepths(
		carrierintel.LookupDepthFull, carrierintel.LookupDepthFull,
	).IgnoreMissing)
	assert.True(t, carrierintel.DiffOptionsForDepths(
		carrierintel.LookupDepthLite, carrierintel.LookupDepthFull,
	).IgnoreMissing)
	assert.True(t, carrierintel.DiffOptionsForDepths(
		carrierintel.LookupDepthFull, carrierintel.LookupDepthLite,
	).IgnoreMissing)
}

func TestDiffProfiles_DepthChangeSkipsFieldsTheShallowerEndpointOmits(t *testing.T) {
	t.Parallel()

	lite := healthyProfile()
	lite.Safety.Rating = ""
	lite.Safety.RatingDate = nil
	lite.Insurance.BIPDRequired = nil
	lite.Identity.DOTAgeDays = nil

	full := healthyProfile()
	full.Safety.RatingDate = new(int64(843609600))
	full.Insurance.BIPDRequired = dec(5_000_000)
	full.Safety.ISSValue = new(69)

	for _, pair := range [][2]*carrierintel.Profile{{lite, full}, {full, lite}} {
		changes, err := carrierintel.DiffProfiles(
			pair[0],
			pair[1],
			carrierintel.DiffOptionsForDepths(
				carrierintel.LookupDepthLite,
				carrierintel.LookupDepthFull,
			),
		)
		require.NoError(t, err)

		assert.Nil(t, changeByPath(changes, "safety.rating"))
		assert.Nil(t, changeByPath(changes, "safety.ratingDate"))
		assert.Nil(t, changeByPath(changes, "insurance.bipdRequired"))

		iss := changeByPath(changes, "safety.issValue")
		require.NotNil(t, iss, "a value present at both depths still produces a change")
		assert.Len(t, changes, 1)
	}

	changes, err := carrierintel.DiffProfiles(lite, full, carrierintel.DiffOptions{})
	require.NoError(t, err)
	require.NotNil(t, changeByPath(changes, "safety.ratingDate"))
	assert.Nil(t, changeByPath(changes, "safety.ratingDate").Prior)
}

func TestDiffProfiles_IgnoresBasicMeasurementDates(t *testing.T) {
	t.Parallel()

	prior := healthyProfile()
	prior.Basics[0].MeasuredAt = new(int64(1_700_000_000))
	current := healthyProfile()
	current.Basics[0].MeasuredAt = new(int64(1_710_000_000))
	current.Basics[0].Violations = new(3)

	changes, err := carrierintel.DiffProfiles(prior, current, carrierintel.DiffOptions{})
	require.NoError(t, err)
	assert.Nil(t, changeByPath(changes, "basics.UnsafeDriving.measuredAt"))
	assert.NotNil(t, changeByPath(changes, "basics.UnsafeDriving.violations"))
}

func TestPlanCarrierSync(t *testing.T) {
	t.Parallel()

	policyID := pulid.MustNew("carins_")
	entity := &carrier.Carrier{
		Name:         "Acme Freight",
		SafetyRating: carrier.SafetyRatingNotRated,
		MCNumber:     "",
		Phone:        "",
		InsurancePolicies: []*carrier.CarrierInsurancePolicy{
			{
				ID:             policyID,
				PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
				PolicyNumber:   "POL-123",
				EffectiveDate:  testNow - 100*timeutils.SecondsPerDay,
				ExpirationDate: testNow + 200*timeutils.SecondsPerDay,
			},
		},
	}

	profile := healthyProfile()
	profile.Contacts = &carrierintel.Contacts{Phone: "(555) 123-4567"}
	cancelAt := testNow + 10*timeutils.SecondsPerDay
	profile.Insurance.Filings = []carrierintel.InsuranceFiling{
		{
			Type:              carrierintel.InsuranceFilingTypeBIPD,
			PolicyNumber:      "pol-123",
			CancelEffectiveAt: &cancelAt,
		},
		{
			Type:         carrierintel.InsuranceFilingTypeCargo,
			PolicyNumber: "CARGO-9",
			InsurerName:  "Great West",
			Coverage:     dec(100_000),
		},
	}

	plan := carrierintel.PlanCarrierSync(entity, profile, carrierintel.SyncSettings{
		AutoApplySafetyRating: true,
		AutoFillFields:        []carrierintel.SyncField{carrierintel.SyncFieldPhone},
	}, testNow)

	autoFields := map[carrierintel.SyncField]string{}
	for _, u := range plan.AutoApply {
		autoFields[u.Field] = u.Proposed
	}
	assert.Equal(t, "Satisfactory", autoFields[carrierintel.SyncFieldSafetyRating])
	assert.Equal(t, "277621", autoFields[carrierintel.SyncFieldMCNumber])
	assert.Equal(t, "5551234567", autoFields[carrierintel.SyncFieldPhone])

	name, ok := plan.Suggestion(carrierintel.SyncFieldName)
	require.True(t, ok)
	assert.Equal(t, "ACME FREIGHT LLC", name.Proposed)

	require.Len(t, plan.InsuranceAutoApply, 1)
	assert.Equal(t, carrierintel.InsuranceChangeShortenExpiration, plan.InsuranceAutoApply[0].Kind)
	assert.Equal(t, cancelAt, *plan.InsuranceAutoApply[0].ProposedExpirationDate)

	require.Len(t, plan.InsuranceSuggestions, 1)
	assert.Equal(t, carrierintel.InsuranceChangeNewFiling, plan.InsuranceSuggestions[0].Kind)
	assert.Equal(
		t,
		carrier.InsurancePolicyTypeCargoLiability,
		plan.InsuranceSuggestions[0].PolicyType,
	)

	applied := carrierintel.ApplyFieldUpdates(entity, plan.AutoApply)
	assert.Len(t, applied, 3)
	assert.Equal(t, carrier.SafetyRatingSatisfactory, entity.SafetyRating)
	assert.Equal(t, 1, carrierintel.ApplyInsuranceChanges(entity, plan.InsuranceAutoApply))
	assert.Equal(t, cancelAt, entity.InsurancePolicies[0].ExpirationDate)
}

func TestPlanCarrierSync_NeverOverwritesWithoutApproval(t *testing.T) {
	t.Parallel()

	entity := &carrier.Carrier{
		Name:         "ACME FREIGHT LLC",
		MCNumber:     "999999",
		Phone:        "5550000000",
		SafetyRating: carrier.SafetyRatingSatisfactory,
	}
	profile := healthyProfile()
	profile.Contacts = &carrierintel.Contacts{Phone: "5551112222"}
	profile.Safety.Rating = carrierintel.SafetyRatingConditional

	plan := carrierintel.PlanCarrierSync(entity, profile, carrierintel.SyncSettings{
		AutoFillFields: []carrierintel.SyncField{carrierintel.SyncFieldPhone},
	}, testNow)

	assert.Empty(t, plan.AutoApply)
	_, hasRating := plan.Suggestion(carrierintel.SyncFieldSafetyRating)
	_, hasMC := plan.Suggestion(carrierintel.SyncFieldMCNumber)
	_, hasPhone := plan.Suggestion(carrierintel.SyncFieldPhone)
	_, hasName := plan.Suggestion(carrierintel.SyncFieldName)
	assert.True(t, hasRating)
	assert.True(t, hasMC)
	assert.True(t, hasPhone)
	assert.False(t, hasName)
}

func TestCapabilitySetBestDepth(t *testing.T) {
	t.Parallel()

	fmcsaOnly := carrierintel.NewCapabilitySet(carrierintel.CapabilityLookupFMCSA)
	depth, ok := fmcsaOnly.BestDepth(carrierintel.LookupDepthFull)
	require.True(t, ok)
	assert.Equal(t, carrierintel.LookupDepthFMCSA, depth)

	all := carrierintel.NewCapabilitySet(
		carrierintel.CapabilityLookupFMCSA,
		carrierintel.CapabilityLookupLite,
		carrierintel.CapabilityLookupFull,
	)
	depth, ok = all.BestDepth(carrierintel.LookupDepthLite)
	require.True(t, ok)
	assert.Equal(t, carrierintel.LookupDepthLite, depth)

	noLite := carrierintel.NewCapabilitySet(
		carrierintel.CapabilityLookupFMCSA,
		carrierintel.CapabilityLookupFull,
	)
	depth, ok = noLite.BestDepth(carrierintel.LookupDepthLite)
	require.True(t, ok)
	assert.Equal(t, carrierintel.LookupDepthFull, depth)

	_, ok = carrierintel.NewCapabilitySet().BestDepth(carrierintel.LookupDepthFMCSA)
	assert.False(t, ok)
}

func TestEnrollmentStateMachine(t *testing.T) {
	t.Parallel()

	e := &carrierintel.CarrierMonitoringEnrollment{
		DesiredState: carrierintel.DesiredStateNotEnrolled,
		VendorState:  carrierintel.VendorStateUnknown,
	}
	assert.True(
		t,
		e.MarkDesired(
			carrierintel.DesiredStateEnrolled,
			carrierintel.EnrollmentReasonManual,
			testNow,
		),
	)
	assert.Equal(t, carrierintel.VendorStatePendingAdd, e.VendorState)
	assert.True(t, e.NeedsSync())

	e.MarkSynced(testNow)
	assert.True(t, e.IsActive())
	assert.True(t, e.OwnedByTrenova)
	assert.False(t, e.NeedsSync())

	assert.False(
		t,
		e.MarkDesired(
			carrierintel.DesiredStateEnrolled,
			carrierintel.EnrollmentReasonManual,
			testNow,
		),
	)
	assert.True(t, e.MarkDesired(carrierintel.DesiredStateNotEnrolled, "", testNow))
	assert.Equal(t, carrierintel.VendorStatePendingRemove, e.VendorState)
	assert.True(t, e.NeedsSync())
	e.MarkSynced(testNow)
	assert.Equal(t, carrierintel.VendorStateRemoved, e.VendorState)

	for range 5 {
		e.MarkFailed("boom", testNow)
	}
	assert.Equal(t, carrierintel.VendorStateFailed, e.VendorState)
}

func TestEquipmentVerificationValidateAndOverride(t *testing.T) {
	t.Parallel()

	v := &carrierintel.CarrierEquipmentVerification{
		CarrierAssignmentID: pulid.MustNew("ca_"),
		CarrierID:           pulid.MustNew("car_"),
		UnitType:            carrierintel.UnitTypeTractor,
		VIN:                 " 1ftfw1et5dfc10312 ",
		PlateNumber:         "abc123",
	}
	multiErr := errortypes.NewMultiError()
	v.Validate(multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.Equal(t, "1FTFW1ET5DFC10312", v.VIN)
	assert.Contains(t, fields, "plateState")

	v.Result = carrierintel.VerificationResultMismatch
	require.Error(t, v.Override(pulid.MustNew("usr_"), " ", testNow))
	require.NoError(
		t,
		v.Override(pulid.MustNew("usr_"), "Swapped tractor, confirmed by phone", testNow),
	)
	assert.True(t, v.IsCleared())
	require.Error(t, v.Override(pulid.MustNew("usr_"), "again", testNow))
}
