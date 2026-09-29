package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRateCoverageAdvisory(t *testing.T) {
	t.Parallel()

	agreementID := pulid.MustNew("ragr_")

	tests := []struct {
		name     string
		entity   *shipment.Shipment
		wantNil  bool
		contains string
	}{
		{
			name: "a shipment the contract priced is fine",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeRated),
				},
			},
			wantNil: true,
		},
		{
			name: "a shipment its formula template priced is fine",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeFormulaFallback),
				},
			},
			wantNil: true,
		},
		{
			name: "a hand-set rate needs no contract",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeManualOverride),
				},
			},
			wantNil: true,
		},
		{
			name: "nothing covered the lane and nothing else could price it",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeNoRateFound),
				},
			},
			contains: "no rating method is set",
		},
		{
			name: "nothing covered the lane but a template did exist",
			entity: &shipment.Shipment{
				FormulaTemplateID: pulid.MustNew("ft_"),
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeNoRateFound),
				},
			},
			contains: "No rate agreement covers this lane",
		},
		{
			name: "a rate that blew up is surfaced with its reason",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source:      string(ratequote.OutcomeError),
					Explanation: "division by zero",
				},
			},
			contains: "division by zero",
		},
		{
			name:     "an unrated shipment with no way to be priced",
			entity:   &shipment.Shipment{},
			contains: "cannot be priced",
		},
		{
			name:    "an unrated shipment carrying a template is left alone",
			entity:  &shipment.Shipment{FormulaTemplateID: pulid.MustNew("ft_")},
			wantNil: true,
		},
		{
			name:    "an unrated shipment carrying a contract is left alone",
			entity:  &shipment.Shipment{RateAgreementID: &agreementID},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			advisory := rateCoverageAdvisory(tt.entity)

			if tt.wantNil {
				assert.Nil(t, advisory)
				return
			}

			require.NotNil(t, advisory)
			assert.Contains(t, advisory.Error(), tt.contains)
			assert.Equal(t, errortypes.SeverityRequireReview, advisory.Severity)
			assert.Equal(t, rateCoverageRuleKey, advisory.RuleKey)
			assert.Equal(t, "freightChargeAmount", advisory.Field)
		})
	}
}

func billingControlWith(
	t *testing.T,
	disposition tenant.UnratedShipmentDisposition,
) *mocks.MockBillingControlRepository {
	t.Helper()

	repo := mocks.NewMockBillingControlRepository(t)
	repo.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(&tenant.BillingControl{UnratedShipmentDisposition: disposition}, nil).
		Maybe()

	return repo
}

func rateCoverageValidator(t *testing.T, billing repositories.BillingControlRepository) *Validator {
	t.Helper()

	v := NewTestValidator(t)
	v.coverage = rateCoverage{billing: billing}
	v.validator = newValidatorBuilder(validatorDeps{
		ControlRepo:    testShipmentControlRepo(t),
		CustomerRepo:   NewTestCustomerRepository(t),
		CommodityRepo:  mocks.NewMockCommodityRepository(t),
		HazmatRuleRepo: mocks.NewMockHazmatSegregationRuleRepository(t),
		ShipmentRepo:   mocks.NewMockShipmentRepository(t),
		BillingRepo:    billing,
	}).Build()

	return v
}

func unratedShipment() *shipment.Shipment {
	entity := validShipmentForValidation()
	entity.FormulaTemplateID = pulid.Nil
	entity.RatingDetail = &shipment.RatingDetail{Source: string(ratequote.OutcomeNoRateFound)}

	return entity
}

func TestRateCoverageRule_RefusesAnUnratedCreate(t *testing.T) {
	t.Parallel()

	for _, disposition := range []tenant.UnratedShipmentDisposition{
		"",
		tenant.UnratedShipmentDispositionFallbackFormulaTemplate,
		tenant.UnratedShipmentDispositionBlock,
	} {
		t.Run(string(disposition), func(t *testing.T) {
			t.Parallel()

			v := rateCoverageValidator(t, billingControlWith(t, disposition))

			multiErr, _ := v.ValidateCreateWithAdvisories(t.Context(), unratedShipment())

			require.NotNil(t, multiErr)
			assertErrorField(t, multiErr, "formulaTemplateId")
			assert.Contains(t, multiErr.Error(), "Choose a rating method")
		})
	}
}

func TestRateCoverageRule_RefusesAShipmentThatCannotBePriced(t *testing.T) {
	t.Parallel()

	v := rateCoverageValidator(t, billingControlWith(t, ""))
	entity := validShipmentForValidation()
	entity.FormulaTemplateID = pulid.Nil

	multiErr, _ := v.ValidateCreateWithAdvisories(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "formulaTemplateId")
}

func TestRateCoverageRule_RefusesWithoutABillingControl(t *testing.T) {
	t.Parallel()

	failing := mocks.NewMockBillingControlRepository(t)
	failing.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("billing control not found"))

	multiErr, _ := rateCoverageValidator(t, failing).
		ValidateCreateWithAdvisories(t.Context(), unratedShipment())

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "formulaTemplateId")
}

func TestRateCoverageRule_ZeroAndFlagKeepsTheShipmentAndFlagsIt(t *testing.T) {
	t.Parallel()

	v := rateCoverageValidator(
		t, billingControlWith(t, tenant.UnratedShipmentDispositionZeroAndFlag),
	)

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), unratedShipment())

	assert.Nil(t, multiErr)
	require.Len(t, advisories, 1)
	assert.Equal(t, rateCoverageRuleKey, advisories[0].RuleKey)
}

func TestRateCoverageRule_APricedCreateNeedsNoControl(t *testing.T) {
	t.Parallel()

	v := rateCoverageValidator(t, mocks.NewMockBillingControlRepository(t))
	entity := validShipmentForValidation()
	entity.RatingDetail = &shipment.RatingDetail{Source: string(ratequote.OutcomeFormulaFallback)}

	multiErr, advisories := v.ValidateCreateWithAdvisories(t.Context(), entity)

	assert.Nil(t, multiErr)
	assert.Empty(t, advisories)
}

func TestRateCoverageRule_AnUpdateThatUnpricesAShipmentIsRefused(t *testing.T) {
	t.Parallel()

	v := rateCoverageValidator(t, billingControlWith(t, ""))
	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.RatingDetail = &shipment.RatingDetail{
		Source: string(ratequote.OutcomeFormulaFallback),
	}
	updated := unratedShipment()
	updated.ID = original.ID
	updated.OrganizationID = original.OrganizationID
	updated.BusinessUnitID = original.BusinessUnitID

	multiErr, _ := v.ValidateUpdateWithOriginalAndAdvisories(t.Context(), original, updated)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "formulaTemplateId")
}

func TestRateCoverageRule_AnAlreadyUnratedShipmentStaysEditable(t *testing.T) {
	t.Parallel()

	v := rateCoverageValidator(t, mocks.NewMockBillingControlRepository(t))
	original := unratedShipment()
	original.ID = pulid.MustNew("shp_")
	updated := unratedShipment()
	updated.ID = original.ID
	updated.OrganizationID = original.OrganizationID
	updated.BusinessUnitID = original.BusinessUnitID

	multiErr, advisories := v.ValidateUpdateWithOriginalAndAdvisories(
		t.Context(), original, updated,
	)

	assert.Nil(t, multiErr, "a shipment unrated before the rule is not locked by it")
	require.Len(t, advisories, 1)
	assert.Equal(t, rateCoverageRuleKey, advisories[0].RuleKey)
}

func TestUnratedCopyVerdict(t *testing.T) {
	t.Parallel()

	agreementID := pulid.MustNew("ragr_")
	tests := []struct {
		name    string
		source  *shipment.Shipment
		unrated bool
	}{
		{
			name: "a source nothing priced",
			source: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{Source: string(ratequote.OutcomeNoRateFound)},
			},
			unrated: true,
		},
		{
			name:    "a source with no rating method and no agreement",
			source:  &shipment.Shipment{},
			unrated: true,
		},
		{
			name: "a source its contract priced",
			source: &shipment.Shipment{
				RateAgreementID: &agreementID,
				RatingDetail:    &shipment.RatingDetail{Source: string(ratequote.OutcomeRated)},
			},
		},
		{
			name:   "a source carrying a rating method",
			source: &shipment.Shipment{FormulaTemplateID: pulid.MustNew("fmt_")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			copied := &shipment.Shipment{FormulaTemplateID: tt.source.FormulaTemplateID}
			verdict := unratedCopyVerdict(tt.source, copied)
			if !tt.unrated {
				assert.Equal(t, priced, verdict)
				return
			}
			assert.NotEqual(t, priced, verdict)
		})
	}
}
