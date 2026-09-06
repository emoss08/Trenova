package driverpay_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validPlan() *driverpay.BenefitPlan {
	return &driverpay.BenefitPlan{
		Status:            domaintypes.StatusActive,
		Code:              "MED-PPO",
		Name:              "Medical PPO",
		PlanType:          driverpay.BenefitPlanMedical,
		PayCodeID:         "pcod_1",
		PlanYear:          2026,
		EmployeeCostMinor: 12_000,
		EmployerCostMinor: 48_000,
	}
}

func TestBenefitPlan_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a complete plan passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		validPlan().Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	// A deduction nobody can categorise is a deduction nobody can explain when
	// it turns up on a settlement.
	t.Run("a plan needs a pay code", func(t *testing.T) {
		t.Parallel()
		entity := validPlan()
		entity.PayCodeID = ""

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "pay code is required")
	})

	// A plan that costs nobody anything would produce a zero deduction on
	// every settlement forever.
	t.Run("a plan has to cost somebody something", func(t *testing.T) {
		t.Parallel()
		entity := validPlan()
		entity.EmployeeCostMinor = 0
		entity.EmployerCostMinor = 0

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "has to cost somebody something")
	})

	// An employer-paid plan is a real thing — the worker contributes nothing
	// and the cover still has a price on it.
	t.Run("an employer-paid plan is allowed", func(t *testing.T) {
		t.Parallel()
		entity := validPlan()
		entity.EmployeeCostMinor = 0

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})
}

func TestBenefitPlan_CostForTier(t *testing.T) {
	t.Parallel()

	plan := validPlan()
	assert.Equal(t, int64(12_000), plan.CostForTier(driverpay.TierEmployee))
	assert.Equal(t, int64(21_600), plan.CostForTier(driverpay.TierEmployeeSpouse))
	assert.Equal(t, int64(19_200), plan.CostForTier(driverpay.TierEmployeeChildren))
	assert.Equal(t, int64(31_200), plan.CostForTier(driverpay.TierFamily))

	// An employer-paid plan scales to nothing rather than to a rounding error.
	free := validPlan()
	free.EmployeeCostMinor = 0
	assert.Equal(t, int64(0), free.CostForTier(driverpay.TierFamily))
}

func validEnrollment() *driverpay.WorkerBenefitEnrollment {
	return &driverpay.WorkerBenefitEnrollment{
		WorkerID:          "wrk_1",
		BenefitPlanID:     "bplan_1",
		Status:            driverpay.EnrollmentActive,
		CoverageTier:      driverpay.TierEmployee,
		EffectiveFrom:     1_800_000_000,
		EmployeeCostMinor: 12_000,
		EmployerCostMinor: 48_000,
	}
}

func TestWorkerBenefitEnrollment_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a complete enrollment passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		validEnrollment().Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	// "Declined" and "nobody asked" are different facts, and only one of them
	// is a problem at audit — so a waiver has to say why.
	t.Run("a waiver needs a reason", func(t *testing.T) {
		t.Parallel()
		entity := validEnrollment()
		entity.Status = driverpay.EnrollmentWaived

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "why the cover was declined")
	})

	t.Run("cover cannot end before it begins", func(t *testing.T) {
		t.Parallel()
		entity := validEnrollment()
		before := entity.EffectiveFrom - 1
		entity.EffectiveTo = &before

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "cannot end before it begins")
	})
}

// A waiver and an ended enrollment are both records of coverage that is not
// happening, and neither should reach a settlement.
func TestBenefitEnrollmentStatus_Deducts(t *testing.T) {
	t.Parallel()

	assert.True(t, driverpay.EnrollmentActive.Deducts())
	assert.False(t, driverpay.EnrollmentPending.Deducts())
	assert.False(t, driverpay.EnrollmentWaived.Deducts())
	assert.False(t, driverpay.EnrollmentEnded.Deducts())
}

func TestWorkerBenefitEnrollment_IsOpen(t *testing.T) {
	t.Parallel()

	start := int64(1_800_000_000)
	end := start + 90*86400

	entity := validEnrollment()
	entity.EffectiveTo = &end
	assert.True(t, entity.IsOpen(start+86400))
	assert.False(t, entity.IsOpen(start-1))
	assert.False(t, entity.IsOpen(end+1))

	// Open cover with no end date runs until somebody closes it.
	open := validEnrollment()
	assert.True(t, open.IsOpen(start+365*86400))

	ended := validEnrollment()
	ended.Status = driverpay.EnrollmentEnded
	assert.False(t, ended.IsOpen(start+86400))
}

// A court order outranks a voluntary deduction, and a benefit somebody chose
// comes last. The order matters when pay will not cover everything.
func TestDeductionKind_DefaultPriority(t *testing.T) {
	t.Parallel()

	assert.Less(
		t,
		driverpay.DeductionKindGarnishment.DefaultPriority(),
		driverpay.DeductionKindStandard.DefaultPriority(),
	)
	assert.Less(
		t,
		driverpay.DeductionKindStandard.DefaultPriority(),
		driverpay.DeductionKindBenefit.DefaultPriority(),
	)
}

func TestRecurringDeduction_GarnishmentNeedsItsOrder(t *testing.T) {
	t.Parallel()

	entity := &driverpay.RecurringDeduction{
		WorkerID:     "wrk_1",
		PayCodeID:    "pcod_1",
		Status:       driverpay.DeductionStatusActive,
		Kind:         driverpay.DeductionKindGarnishment,
		Frequency:    driverpay.DeductionFrequencyEverySettlement,
		Description:  "Child support",
		AmountMinor:  25_000,
		StartDate:    1_800_000_000,
		CurrencyCode: "USD",
		Priority:     10,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "needs the order it is taken under")

	entity.CourtOrderNumber = "CV-2026-1188"
	multiErr = errortypes.NewMultiError()
	entity.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}
