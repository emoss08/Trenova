package shipmentcommercial

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fuelsurchargeportsmock"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial/shipmentcommercialtest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAdoptContractRate_AppliesAgreementAccessorials(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	entity := ratedShipment(agreement.ID)

	calculator := agreementCalculator(t, agreement, nil)

	adoptContract(t, calculator, entity)
	require.NoError(
		t,
		calculator.Recalculate(
			t.Context(),
			entity,
			&tenant.ShipmentControl{},
			pulid.MustNew("usr_"),
		),
	)

	require.Len(t, entity.AdditionalCharges, 1)
	charge := entity.AdditionalCharges[0]

	assert.Equal(t, shipment.SystemOwnerAgreement, charge.Owner())
	assert.Equal(t, agreement.Accessorials[0].ID, *charge.RateAgreementAccessorialID)
	assert.True(t, decimal.NewFromInt(75).Equal(charge.Amount))
	assert.True(t, decimal.NewFromInt(175).Equal(entity.TotalChargeAmount.Decimal))
}

// Auto-rating can be asked for again, so the same contract must not add its
// accessorial a second time when it does.
func TestAdoptContractRate_AgreementAccessorialsAreIdempotent(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	entity := ratedShipment(agreement.ID)

	calculator := agreementCalculator(t, agreement, nil)

	adoptContract(t, calculator, entity)
	require.Len(t, entity.AdditionalCharges, 1)

	first := entity.AdditionalCharges[0]
	first.ID = pulid.MustNew("ac_")

	adoptContract(t, calculator, entity)

	require.Len(t, entity.AdditionalCharges, 1)
	assert.Equal(t, first.ID, entity.AdditionalCharges[0].ID,
		"the existing charge should be reused so its audit trail survives")
}

// The whole point of applying a contract once is that the shipment's own
// charge rows are then the truth. A recalculation runs from seven call sites —
// a stop edit, an assignment, a fuel price job — and any of them rebuilding the
// contract's rows would silently undo a figure a rater has since agreed.
func TestRecalculate_LeavesContractAccessorialsAlone(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	entity := ratedShipment(agreement.ID)
	entity.AutoRated = true

	calculator := agreementCalculator(t, agreement, nil)
	adoptContract(t, calculator, entity)
	require.Len(t, entity.AdditionalCharges, 1)

	// A rater renegotiates the contract's charge downwards.
	entity.AdditionalCharges[0].ID = pulid.MustNew("ac_")
	entity.AdditionalCharges[0].Amount = decimal.NewFromInt(25)

	require.NoError(
		t,
		calculator.Recalculate(
			t.Context(),
			entity,
			&tenant.ShipmentControl{},
			pulid.MustNew("usr_"),
		),
	)

	require.Len(t, entity.AdditionalCharges, 1)
	assert.True(t, decimal.NewFromInt(25).Equal(entity.AdditionalCharges[0].Amount),
		"a recalculation must not reprice a charge somebody has since agreed")
}

// A charge whose contract row no longer applies has to disappear, otherwise a
// waived accessorial keeps billing forever.
func TestAdoptContractRate_RemovesAgreementChargeWhenNoLongerApplicable(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	entity := ratedShipment(agreement.ID)
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			ID:                         pulid.MustNew("ac_"),
			AccessorialChargeID:        agreement.Accessorials[0].AccessorialChargeID,
			IsSystemGenerated:          true,
			Method:                     accessorialcharge.MethodFlat,
			Amount:                     decimal.NewFromInt(75),
			Unit:                       1,
			RateAgreementAccessorialID: &agreement.Accessorials[0].ID,
		},
	}

	agreement.Accessorials[0].Waived = true

	calculator := agreementCalculator(t, agreement, nil)

	adoptContract(t, calculator, entity)

	assert.Empty(t, entity.AdditionalCharges)
}

func TestAdoptContractRate_SkipsAccessorialWhoseConditionIsFalse(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	agreement.Accessorials[0].ApplyCondition = "totalStops > 2"
	entity := ratedShipment(agreement.ID)

	predicate := formulatemplateportsmock.NewMockFormulaPredicateEvaluator(t)
	predicate.EXPECT().
		EvaluatePredicates(mock.Anything, mock.AnythingOfType("*formulatemplateports.EvaluatePredicatesRequest")).
		Return([]formulatemplateports.PredicateOutcome{{Holds: false}}, nil).
		Once()

	calculator := agreementCalculator(t, agreement, predicate)

	adoptContract(t, calculator, entity)

	assert.Empty(t, entity.AdditionalCharges)
}

func TestAdoptContractRate_EvaluatesEveryConditionInOneCall(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	agreement.Accessorials[0].ApplyCondition = "totalStops > 2"
	second := *agreement.Accessorials[0]
	second.ID = pulid.MustNew("raga_")
	second.AccessorialChargeID = pulid.MustNew("acc_")
	second.ApplyCondition = "hasHazmat"
	unconditional := *agreement.Accessorials[0]
	unconditional.ID = pulid.MustNew("raga_")
	unconditional.AccessorialChargeID = pulid.MustNew("acc_")
	unconditional.ApplyCondition = ""
	agreement.Accessorials = append(agreement.Accessorials, &second, &unconditional)
	entity := ratedShipment(agreement.ID)

	predicate := formulatemplateportsmock.NewMockFormulaPredicateEvaluator(t)
	predicate.EXPECT().
		EvaluatePredicates(mock.Anything, mock.MatchedBy(
			func(req *formulatemplateports.EvaluatePredicatesRequest) bool {
				return assert.ObjectsAreEqual(
					[]string{"totalStops > 2", "hasHazmat"},
					req.Expressions,
				) && req.SchemaID == shipmentSchemaID
			},
		)).
		Return([]formulatemplateports.PredicateOutcome{
			{Holds: false},
			{Holds: true},
		}, nil).
		Once()

	calculator := agreementCalculator(t, agreement, predicate)

	adoptContract(t, calculator, entity)

	applied := make([]pulid.ID, 0, len(entity.AdditionalCharges))
	for _, charge := range entity.AdditionalCharges {
		applied = append(applied, *charge.RateAgreementAccessorialID)
	}
	assert.ElementsMatch(t, []pulid.ID{second.ID, unconditional.ID}, applied)
}

// Charging a customer for something the contract may not entitle us to is the
// more expensive of the two mistakes, so an unevaluable condition withholds the
// charge rather than applying it.
func TestAdoptContractRate_WithholdsAccessorialWhenConditionErrors(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	agreement.Accessorials[0].ApplyCondition = "nonsense("
	entity := ratedShipment(agreement.ID)

	predicate := formulatemplateportsmock.NewMockFormulaPredicateEvaluator(t)
	predicate.EXPECT().
		EvaluatePredicates(mock.Anything, mock.AnythingOfType("*formulatemplateports.EvaluatePredicatesRequest")).
		Return([]formulatemplateports.PredicateOutcome{{Err: errors.New("does not evaluate")}}, nil).
		Once()

	calculator := agreementCalculator(t, agreement, predicate)

	adoptContract(t, calculator, entity)

	assert.Empty(t, entity.AdditionalCharges)
}

func TestSyncFuelSurcharge_PassesAgreementFuelBinding(t *testing.T) {
	t.Parallel()

	programID := pulid.MustNew("fsp_")
	agreement := agreementWithAccessorial(t, decimal.NewFromInt(0))
	agreement.Accessorials = nil
	agreement.FuelBinding = &rateagreement.RateAgreementFuelBinding{
		ID:                     pulid.MustNew("ragf_"),
		FuelSurchargeProgramID: programID,
		CapAmount:              decimal.NewNullDecimal(decimal.NewFromInt(200)),
	}

	entity := ratedShipment(agreement.ID)

	var captured *fuelsurchargeports.FuelProgramOverride

	fuel := fuelsurchargeportsmock.NewMockFuelSurchargeResolver(t)
	fuel.EXPECT().
		ResolveShipmentCharge(
			mock.Anything,
			mock.AnythingOfType("*fuelsurchargeports.ResolveShipmentChargeRequest"),
		).
		RunAndReturn(func(
			_ context.Context,
			req *fuelsurchargeports.ResolveShipmentChargeRequest,
		) (*fuelsurchargeports.ResolvedFuelSurcharge, error) {
			captured = req.Override
			return nil, nil
		}).
		Once()

	calculator := agreementCalculator(t, agreement, nil)
	calculator.fuelSurcharge = fuel

	require.NoError(
		t,
		calculator.Recalculate(
			t.Context(),
			entity,
			&tenant.ShipmentControl{},
			pulid.MustNew("usr_"),
		),
	)

	require.NotNil(t, captured)
	assert.Equal(t, programID, captured.ProgramID)
	assert.Equal(t, agreement.ID, captured.AgreementID)
	assert.True(t, captured.AppliesTerms())
	assert.True(t, decimal.NewFromInt(200).Equal(captured.CapAmount.Decimal))
}

func TestFuelOverride_IsNilWithoutABinding(t *testing.T) {
	t.Parallel()

	assert.Nil(t, fuelOverride(nil))
	assert.Nil(t, fuelOverride(&rateagreement.RateAgreement{}))
}

// adoptContract runs the one step that seats a contract on a shipment, which is
// where the accessorial reconciliation lives.
func adoptContract(t *testing.T, calculator *Calculator, entity *shipment.Shipment) {
	t.Helper()

	rated, err := calculator.RateAgainstContract(t.Context(), entity, pulid.MustNew("usr_"), false)
	require.NoError(t, err)

	calculator.AdoptContractRate(t.Context(), entity, rated)
}

func agreementWithAccessorial(t *testing.T, amount decimal.Decimal) *rateagreement.RateAgreement {
	t.Helper()

	agreementID := pulid.MustNew("ragr_")

	return &rateagreement.RateAgreement{
		ID:     agreementID,
		Status: rateagreement.StatusActive,
		Accessorials: []*rateagreement.RateAgreementAccessorial{
			{
				ID:                  pulid.MustNew("raga_"),
				RateAgreementID:     agreementID,
				AccessorialChargeID: pulid.MustNew("acc_"),
				Method:              accessorialcharge.MethodFlat,
				Amount:              amount,
				AutoApply:           true,
			},
		},
	}
}

func ratedShipment(agreementID pulid.ID) *shipment.Shipment {
	entity := validShipment()
	entity.RateAgreementID = &agreementID

	return entity
}

func agreementCalculator(
	t *testing.T,
	agreement *rateagreement.RateAgreement,
	predicate formulatemplateports.FormulaPredicateEvaluator,
) *Calculator {
	t.Helper()

	calculator := New(Params{
		Logger:        zap.NewNop(),
		RateEngine:    shipmentcommercialtest.StubRateEngineForAgreement(t, 100, &agreement.ID),
		Predicate:     predicate,
		AgreementRepo: agreementRepoFor(t, agreement),
	})
	calculator.now = func() int64 { return shipmentcommercialtest.RatedAt }

	return calculator
}
