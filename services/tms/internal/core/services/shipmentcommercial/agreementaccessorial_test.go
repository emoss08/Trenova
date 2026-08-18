package shipmentcommercial

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRecalculate_AppliesAgreementAccessorials(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	entity := ratedShipment(agreement.ID)

	calculator := agreementCalculator(t, agreement, nil)

	require.NoError(t, calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")))

	require.Len(t, entity.AdditionalCharges, 1)
	charge := entity.AdditionalCharges[0]

	assert.Equal(t, shipment.SystemOwnerAgreement, charge.Owner())
	assert.Equal(t, agreement.Accessorials[0].ID, *charge.RateAgreementAccessorialID)
	assert.True(t, decimal.NewFromInt(75).Equal(charge.Amount))
	assert.True(t, decimal.NewFromInt(175).Equal(entity.TotalChargeAmount.Decimal))
}

// A shipment is recalculated from seven different call sites, so the same
// contract must not add its accessorial again on every one of them.
func TestRecalculate_AgreementAccessorialsAreIdempotent(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	entity := ratedShipment(agreement.ID)

	calculator := agreementCalculator(t, agreement, nil)
	control := &tenant.ShipmentControl{}
	userID := pulid.MustNew("usr_")

	require.NoError(t, calculator.Recalculate(t.Context(), entity, control, userID))
	require.Len(t, entity.AdditionalCharges, 1)

	first := entity.AdditionalCharges[0]
	first.ID = pulid.MustNew("ac_")

	require.NoError(t, calculator.Recalculate(t.Context(), entity, control, userID))

	require.Len(t, entity.AdditionalCharges, 1)
	assert.Equal(t, first.ID, entity.AdditionalCharges[0].ID,
		"the existing charge should be reused so its audit trail survives")
}

// A charge whose contract row no longer applies has to disappear, otherwise a
// waived accessorial keeps billing forever.
func TestRecalculate_RemovesAgreementChargeWhenNoLongerApplicable(t *testing.T) {
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

	require.NoError(t, calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")))

	assert.Empty(t, entity.AdditionalCharges)
}

func TestRecalculate_SkipsAccessorialWhoseConditionIsFalse(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	agreement.Accessorials[0].ApplyCondition = "totalStops > 2"
	entity := ratedShipment(agreement.ID)

	predicate := mocks.NewMockFormulaPredicateEvaluator(t)
	predicate.EXPECT().
		EvaluatePredicate(mock.Anything, mock.AnythingOfType("*services.EvaluatePredicateRequest")).
		Return(false, nil).
		Once()

	calculator := agreementCalculator(t, agreement, predicate)

	require.NoError(t, calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")))

	assert.Empty(t, entity.AdditionalCharges)
}

// Charging a customer for something the contract may not entitle us to is the
// more expensive of the two mistakes, so an unevaluable condition withholds the
// charge rather than applying it.
func TestRecalculate_WithholdsAccessorialWhenConditionErrors(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	agreement.Accessorials[0].ApplyCondition = "nonsense("
	entity := ratedShipment(agreement.ID)

	predicate := mocks.NewMockFormulaPredicateEvaluator(t)
	predicate.EXPECT().
		EvaluatePredicate(mock.Anything, mock.AnythingOfType("*services.EvaluatePredicateRequest")).
		Return(false, errors.New("compile failed")).
		Once()

	calculator := agreementCalculator(t, agreement, predicate)

	require.NoError(t, calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")))

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

	var captured *services.FuelProgramOverride

	fuel := mocks.NewMockFuelSurchargeResolver(t)
	fuel.EXPECT().
		ResolveShipmentCharge(
			mock.Anything,
			mock.AnythingOfType("*services.ResolveShipmentChargeRequest"),
		).
		RunAndReturn(func(
			_ context.Context,
			req *services.ResolveShipmentChargeRequest,
		) (*services.ResolvedFuelSurcharge, error) {
			captured = req.Override
			return nil, nil
		}).
		Once()

	calculator := agreementCalculator(t, agreement, nil)
	calculator.fuelSurcharge = fuel

	require.NoError(t, calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")))

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
	predicate services.FormulaPredicateEvaluator,
) *Calculator {
	t.Helper()

	agreementRepo := mocks.NewMockRateAgreementRepository(t)
	agreementRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetRateAgreementByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.GetRateAgreementByIDRequest,
		) (*rateagreement.RateAgreement, error) {
			if req.RateAgreementID != agreement.ID {
				return nil, errors.New("unexpected agreement id")
			}
			return agreement, nil
		}).
		Maybe()

	calculator := New(Params{
		Logger:        zap.NewNop(),
		RateEngine:    StubRateEngineForAgreement(t, 100, &agreement.ID),
		Predicate:     predicate,
		AgreementRepo: agreementRepo,
	})
	calculator.now = func() int64 { return ratedAt }

	return calculator
}
