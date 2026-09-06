package worker

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldsOf(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	return fields
}

func TestComputePTODays(t *testing.T) {
	t.Parallel()
	ny, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	mon := time.Date(2026, time.March, 2, 9, 0, 0, 0, ny)
	sun := time.Date(2026, time.March, 8, 17, 0, 0, 0, ny)

	assert.True(t, ComputePTODays(mon.Unix(), sun.Unix(), ny, true, nil).Equal(decimal.NewFromInt(7)))
	assert.True(t, ComputePTODays(mon.Unix(), sun.Unix(), ny, false, nil).Equal(decimal.NewFromInt(5)))
	assert.True(t, ComputePTODays(mon.Unix(), mon.Add(time.Hour).Unix(), ny, false, nil).Equal(decimal.NewFromInt(1)),
		"same-day request counts as one day")
	assert.True(t, ComputePTODays(sun.Unix(), mon.Unix(), ny, true, nil).IsZero(), "reversed range is zero")

	dstStart := time.Date(2026, time.March, 7, 23, 30, 0, 0, ny)
	dstEnd := time.Date(2026, time.March, 9, 0, 30, 0, 0, ny)
	assert.True(t, ComputePTODays(dstStart.Unix(), dstEnd.Unix(), ny, true, nil).Equal(decimal.NewFromInt(3)),
		"DST transition does not drop a day")

	utcEveningStart := time.Date(2026, time.March, 2, 23, 30, 0, 0, time.UTC)
	assert.True(t, ComputePTODays(utcEveningStart.Unix(), utcEveningStart.Unix(), time.UTC, true, nil).Equal(decimal.NewFromInt(1)))
	assert.True(t, ComputePTODays(utcEveningStart.Unix(), utcEveningStart.Unix(), nil, true, nil).Equal(decimal.NewFromInt(1)),
		"nil location falls back to UTC")
}

func validPolicy() *PTOPolicy {
	return &PTOPolicy{
		Name:          "Standard",
		Code:          "STD",
		Status:        PTOPolicyStatusActive,
		YearBasis:     PTOYearBasisCalendarYear,
		CountWeekends: true,
		Rules: []*PTOPolicyRule{
			{
				PTOType:           PTOTypeVacation,
				AccrualMethod:     PTOAccrualMethodMonthly,
				AccrualAmountDays: decimal.RequireFromString("0.83"),
			},
		},
	}
}

func TestPTOPolicyValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		validPolicy().Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), "%v", multiErr.Errors)
	})

	t.Run("negative balances need a floor", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.AllowNegative = true
		multiErr := errortypes.NewMultiError()
		p.Validate(multiErr)
		assert.Contains(t, fieldsOf(multiErr), "negativeFloorDays")
	})

	t.Run("floor without allow negative is rejected", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.NegativeFloorDays = decimal.NewFromInt(-3)
		multiErr := errortypes.NewMultiError()
		p.Validate(multiErr)
		assert.Contains(t, fieldsOf(multiErr), "negativeFloorDays")
	})

	t.Run("duplicate rule types", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.Rules = append(p.Rules, &PTOPolicyRule{
			PTOType:       PTOTypeVacation,
			AccrualMethod: PTOAccrualMethodNone,
		})
		multiErr := errortypes.NewMultiError()
		p.Validate(multiErr)
		assert.Contains(t, fieldsOf(multiErr), "rules[1].ptoType")
	})

	t.Run("accruing rule needs an amount", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.Rules[0].AccrualAmountDays = decimal.Zero
		multiErr := errortypes.NewMultiError()
		p.Validate(multiErr)
		assert.Contains(t, fieldsOf(multiErr), "rules[0].accrualAmountDays")
	})

	t.Run("expiry needs a carryover cap", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.Rules[0].CarryoverExpiryDays = 30
		multiErr := errortypes.NewMultiError()
		p.Validate(multiErr)
		assert.Contains(t, fieldsOf(multiErr), "rules[0].carryoverExpiryDays")
	})

	assert.Same(t, validPolicy().RuleFor(PTOTypeSick), (*PTOPolicyRule)(nil))
}

func TestLedgerEntrySignRules(t *testing.T) {
	t.Parallel()

	base := func(entryType PTOLedgerEntryType, amount string) *WorkerPTOLedgerEntry {
		return &WorkerPTOLedgerEntry{
			WorkerID:    "wrk_1",
			PTOType:     PTOTypeVacation,
			EntryType:   entryType,
			ActorType:   PTOLedgerActorSystem,
			AmountDays:  decimal.RequireFromString(amount),
			EffectiveAt: 1_700_000_000,
			Note:        "n",
		}
	}

	cases := []struct {
		name   string
		entry  *WorkerPTOLedgerEntry
		fields []string
	}{
		{"accrual must be positive", base(PTOLedgerEntryAccrual, "-1"), []string{"amountDays"}},
		{"usage must be negative", base(PTOLedgerEntryUsage, "1"), []string{"amountDays"}},
		{"zero is never allowed", base(PTOLedgerEntryAdjustment, "0"), []string{"amountDays"}},
		{"adjustment either sign", base(PTOLedgerEntryAdjustment, "-2.5"), nil},
		{"expiry negative", base(PTOLedgerEntryExpiry, "-2"), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			multiErr := errortypes.NewMultiError()
			tc.entry.Validate(multiErr)
			if tc.fields == nil {
				assert.False(t, multiErr.HasErrors(), "%v", multiErr.Errors)
				return
			}
			for _, f := range tc.fields {
				assert.Contains(t, fieldsOf(multiErr), f)
			}
		})
	}

	adjustment := base(PTOLedgerEntryAdjustment, "1")
	adjustment.Note = ""
	multiErr := errortypes.NewMultiError()
	adjustment.Validate(multiErr)
	assert.Contains(t, fieldsOf(multiErr), "note")
}

func TestBalanceApplyKeepsRunningTotals(t *testing.T) {
	t.Parallel()

	bal := &WorkerPTOBalance{BalanceDays: decimal.NewFromInt(3), EntryCount: 4}

	accrual := &WorkerPTOLedgerEntry{EntryType: PTOLedgerEntryAccrual, AmountDays: decimal.NewFromInt(2)}
	bal.Apply(accrual)
	assert.EqualValues(t, 5, accrual.Sequence)
	assert.True(t, accrual.BalanceAfterDays.Equal(decimal.NewFromInt(5)))
	assert.True(t, bal.AccruedYTDDays.Equal(decimal.NewFromInt(2)))

	usage := &WorkerPTOLedgerEntry{EntryType: PTOLedgerEntryUsage, AmountDays: decimal.NewFromInt(-4)}
	bal.Apply(usage)
	assert.EqualValues(t, 6, usage.Sequence)
	assert.True(t, bal.BalanceDays.Equal(decimal.NewFromInt(1)))
	assert.True(t, bal.UsedYTDDays.Equal(decimal.NewFromInt(4)))

	reversal := &WorkerPTOLedgerEntry{EntryType: PTOLedgerEntryReversal, AmountDays: decimal.NewFromInt(4)}
	bal.Apply(reversal)
	assert.True(t, bal.BalanceDays.Equal(decimal.NewFromInt(5)))
	assert.True(t, bal.UsedYTDDays.IsZero())

	assignment := &WorkerPTOPolicyAssignment{EffectiveFrom: 100}
	assert.True(t, assignment.IsOpen())
	assert.True(t, assignment.CoversDate(100))
	assert.False(t, assignment.CoversDate(99))
	end := int64(200)
	assignment.EffectiveTo = &end
	assert.False(t, assignment.CoversDate(200))
	assert.True(t, assignment.CoversDate(199))
}
