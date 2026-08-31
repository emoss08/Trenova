package driverpay

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validRecurringDeduction() RecurringDeduction {
	return RecurringDeduction{
		ID:             pulid.MustNew("rded_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		WorkerID:       pulid.MustNew("wrk_"),
		PayCodeID:      pulid.MustNew("payc_"),
		Status:         DeductionStatusActive,
		Frequency:      DeductionFrequencyEverySettlement,
		Description:    "Truck lease payment",
		AmountMinor:    45_000,
		StartDate:      1_700_000_000,
		CurrencyCode:   "USD",
	}
}

func validRecurringEarning() RecurringEarning {
	return RecurringEarning{
		ID:             pulid.MustNew("rern_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		WorkerID:       pulid.MustNew("wrk_"),
		PayCodeID:      pulid.MustNew("payc_"),
		Status:         EarningStatusActive,
		Frequency:      EarningFrequencyEverySettlement,
		Description:    "Longevity bonus",
		AmountMinor:    10_000,
		StartDate:      1_700_000_000,
		CurrencyCode:   "USD",
	}
}

func TestRecurringDeduction_Validate(t *testing.T) {
	t.Parallel()

	invalidCap := int64(0)
	validCap := int64(500_000)
	endBeforeStart := int64(1_600_000_000)
	endAfterStart := int64(1_800_000_000)

	tests := []struct {
		name    string
		mutate  func(r *RecurringDeduction)
		wantErr bool
	}{
		{name: "valid deduction passes", mutate: func(r *RecurringDeduction) {}, wantErr: false},
		{
			name:    "missing worker fails",
			mutate:  func(r *RecurringDeduction) { r.WorkerID = "" },
			wantErr: true,
		},
		{
			name:    "missing pay code fails",
			mutate:  func(r *RecurringDeduction) { r.PayCodeID = "" },
			wantErr: true,
		},
		{
			name:    "missing description fails",
			mutate:  func(r *RecurringDeduction) { r.Description = "" },
			wantErr: true,
		},
		{
			name:    "invalid status fails",
			mutate:  func(r *RecurringDeduction) { r.Status = "Suspended" },
			wantErr: true,
		},
		{
			name:    "invalid frequency fails",
			mutate:  func(r *RecurringDeduction) { r.Frequency = "Weekly" },
			wantErr: true,
		},
		{
			name:    "non positive amount fails",
			mutate:  func(r *RecurringDeduction) { r.AmountMinor = 0 },
			wantErr: true,
		},
		{
			name:    "non positive cap fails",
			mutate:  func(r *RecurringDeduction) { r.TotalCapMinor = &invalidCap },
			wantErr: true,
		},
		{
			name:    "positive cap passes",
			mutate:  func(r *RecurringDeduction) { r.TotalCapMinor = &validCap },
			wantErr: false,
		},
		{
			name:    "end date before start fails",
			mutate:  func(r *RecurringDeduction) { r.EndDate = &endBeforeStart },
			wantErr: true,
		},
		{
			name:    "end date after start passes",
			mutate:  func(r *RecurringDeduction) { r.EndDate = &endAfterStart },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entity := validRecurringDeduction()
			tt.mutate(&entity)
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestRecurringDeduction_CapMath(t *testing.T) {
	t.Parallel()

	t.Run("no cap yields nil remaining and full next amount", func(t *testing.T) {
		t.Parallel()
		deduction := validRecurringDeduction()
		assert.Nil(t, deduction.RemainingCapMinor())
		assert.Equal(t, deduction.AmountMinor, deduction.NextAmountMinor())
	})

	t.Run("remaining cap limits the next amount", func(t *testing.T) {
		t.Parallel()
		cap := int64(100_000)
		deduction := validRecurringDeduction()
		deduction.TotalCapMinor = &cap
		deduction.DeductedToDateMinor = 80_000

		remaining := deduction.RemainingCapMinor()
		assert.NotNil(t, remaining)
		assert.EqualValues(t, 20_000, *remaining)
		assert.EqualValues(t, 20_000, deduction.NextAmountMinor())
	})

	t.Run("exhausted cap clamps at zero", func(t *testing.T) {
		t.Parallel()
		cap := int64(100_000)
		deduction := validRecurringDeduction()
		deduction.TotalCapMinor = &cap
		deduction.DeductedToDateMinor = 120_000

		remaining := deduction.RemainingCapMinor()
		assert.NotNil(t, remaining)
		assert.EqualValues(t, 0, *remaining)
		assert.EqualValues(t, 0, deduction.NextAmountMinor())
	})

	t.Run("inactive deduction yields zero next amount", func(t *testing.T) {
		t.Parallel()
		deduction := validRecurringDeduction()
		deduction.Status = DeductionStatusPaused
		assert.EqualValues(t, 0, deduction.NextAmountMinor())
	})
}

func TestRecurringDeduction_IsEscrowContribution(t *testing.T) {
	t.Parallel()

	deduction := validRecurringDeduction()
	assert.False(t, deduction.IsEscrowContribution())

	escrowID := pulid.MustNew("escr_")
	deduction.EscrowAccountID = &escrowID
	assert.True(t, deduction.IsEscrowContribution())

	nilID := pulid.Nil
	deduction.EscrowAccountID = &nilID
	assert.False(t, deduction.IsEscrowContribution())
}

func TestRecurringEarning_Validate(t *testing.T) {
	t.Parallel()

	invalidCap := int64(-1)
	endBeforeStart := int64(1_600_000_000)

	tests := []struct {
		name    string
		mutate  func(r *RecurringEarning)
		wantErr bool
	}{
		{name: "valid earning passes", mutate: func(r *RecurringEarning) {}, wantErr: false},
		{
			name:    "missing worker fails",
			mutate:  func(r *RecurringEarning) { r.WorkerID = "" },
			wantErr: true,
		},
		{
			name:    "invalid status fails",
			mutate:  func(r *RecurringEarning) { r.Status = "Suspended" },
			wantErr: true,
		},
		{
			name:    "invalid frequency fails",
			mutate:  func(r *RecurringEarning) { r.Frequency = "Weekly" },
			wantErr: true,
		},
		{
			name:    "non positive amount fails",
			mutate:  func(r *RecurringEarning) { r.AmountMinor = -5 },
			wantErr: true,
		},
		{
			name:    "negative cap fails",
			mutate:  func(r *RecurringEarning) { r.TotalCapMinor = &invalidCap },
			wantErr: true,
		},
		{
			name:    "end date before start fails",
			mutate:  func(r *RecurringEarning) { r.EndDate = &endBeforeStart },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entity := validRecurringEarning()
			tt.mutate(&entity)
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestRecurringEarning_CapMath(t *testing.T) {
	t.Parallel()

	t.Run("no cap yields nil remaining and full next amount", func(t *testing.T) {
		t.Parallel()
		earning := validRecurringEarning()
		assert.Nil(t, earning.RemainingCapMinor())
		assert.Equal(t, earning.AmountMinor, earning.NextAmountMinor())
	})

	t.Run("remaining cap limits the next amount", func(t *testing.T) {
		t.Parallel()
		cap := int64(25_000)
		earning := validRecurringEarning()
		earning.TotalCapMinor = &cap
		earning.PaidToDateMinor = 20_000

		remaining := earning.RemainingCapMinor()
		assert.NotNil(t, remaining)
		assert.EqualValues(t, 5_000, *remaining)
		assert.EqualValues(t, 5_000, earning.NextAmountMinor())
	})

	t.Run("exhausted cap clamps at zero", func(t *testing.T) {
		t.Parallel()
		cap := int64(25_000)
		earning := validRecurringEarning()
		earning.TotalCapMinor = &cap
		earning.PaidToDateMinor = 30_000

		remaining := earning.RemainingCapMinor()
		assert.NotNil(t, remaining)
		assert.EqualValues(t, 0, *remaining)
		assert.EqualValues(t, 0, earning.NextAmountMinor())
	})

	t.Run("paused earning yields zero next amount", func(t *testing.T) {
		t.Parallel()
		earning := validRecurringEarning()
		earning.Status = EarningStatusPaused
		assert.EqualValues(t, 0, earning.NextAmountMinor())
	})
}
