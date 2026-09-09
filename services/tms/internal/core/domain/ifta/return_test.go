package ifta_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldErrors(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	return fields
}

func ptrInt64(v int64) *int64 { return &v }

func draftReturn() *ifta.Return {
	return &ifta.Return{
		Year:         2026,
		Quarter:      2,
		Status:       ifta.ReturnStatusDraft,
		Timezone:     "America/New_York",
		PeriodStart:  1_775_016_000,
		PeriodEnd:    1_782_878_400,
		CurrencyCode: "USD",
	}
}

func finalizedReturn() *ifta.Return {
	r := draftReturn()
	r.Status = ifta.ReturnStatusFinalized
	r.FinalizedAt = ptrInt64(1_783_000_000)
	r.FinalizedByID = pulid.MustNew("usr_")
	return r
}

func filedReturn() *ifta.Return {
	r := finalizedReturn()
	r.Status = ifta.ReturnStatusFiled
	r.FiledAt = ptrInt64(1_783_100_000)
	r.FiledByID = pulid.MustNew("usr_")
	r.FilingReference = "TX-2026-Q2-0001"
	return r
}

func TestReturnStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from ifta.ReturnStatus
		to   ifta.ReturnStatus
		want bool
	}{
		{ifta.ReturnStatusDraft, ifta.ReturnStatusFinalized, true},
		{ifta.ReturnStatusDraft, ifta.ReturnStatusFiled, false},
		{ifta.ReturnStatusDraft, ifta.ReturnStatusDraft, false},
		{ifta.ReturnStatusFinalized, ifta.ReturnStatusDraft, true},
		{ifta.ReturnStatusFinalized, ifta.ReturnStatusFiled, true},
		{ifta.ReturnStatusFinalized, ifta.ReturnStatusFinalized, false},
		{ifta.ReturnStatusFiled, ifta.ReturnStatusDraft, false},
		{ifta.ReturnStatusFiled, ifta.ReturnStatusFinalized, false},
		{ifta.ReturnStatusFiled, ifta.ReturnStatusFiled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.from.CanTransitionTo(tt.to))
		})
	}

	assert.False(t, ifta.ReturnStatusDraft.IsLocked())
	assert.True(t, ifta.ReturnStatusFinalized.IsLocked())
	assert.True(t, ifta.ReturnStatusFiled.IsLocked())
}

func TestReturn_GuardsByStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ret         *ifta.Return
		recompute   bool
		finalize    bool
		reopen      bool
		markFiled   bool
		amend       bool
		deleteDraft bool
		locked      bool
	}{
		{"draft", draftReturn(), true, true, false, false, false, true, false},
		{"finalized", finalizedReturn(), false, false, true, true, false, false, true},
		{"filed", filedReturn(), false, false, false, false, true, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.recompute, tt.ret.CanRecompute(), "CanRecompute")
			assert.Equal(t, tt.finalize, tt.ret.CanFinalize(), "CanFinalize")
			assert.Equal(t, tt.reopen, tt.ret.CanReopen(), "CanReopen")
			assert.Equal(t, tt.markFiled, tt.ret.CanMarkFiled(), "CanMarkFiled")
			assert.Equal(t, tt.amend, tt.ret.CanAmend(), "CanAmend")
			assert.Equal(t, tt.deleteDraft, tt.ret.CanDelete(), "CanDelete")
			assert.Equal(t, tt.locked, tt.ret.IsLocked(), "IsLocked")
		})
	}
}

func TestReturn_CanFinalizeDependsOnBlockingProblems(t *testing.T) {
	t.Parallel()

	t.Run("a missing rate blocks", func(t *testing.T) {
		t.Parallel()

		r := draftReturn()
		r.Problems = []ifta.Problem{
			{Code: ifta.ProblemUnattributedMiles, Message: "12 moves have no breakdown"},
			{
				Code:             ifta.ProblemMissingRate,
				Message:          "No rate for TX diesel",
				JurisdictionCode: "TX",
				FuelType:         domaintypes.IFTAFuelTypeDiesel,
			},
		}

		assert.False(t, r.CanFinalize())
		assert.True(t, r.HasProblem(ifta.ProblemMissingRate))
		assert.Len(t, r.BlockingProblems(), 1)
	})

	t.Run("unattributed miles only warn", func(t *testing.T) {
		t.Parallel()

		r := draftReturn()
		r.Problems = []ifta.Problem{
			{Code: ifta.ProblemUnattributedMiles, Message: "12 moves have no breakdown"},
			{Code: ifta.ProblemNonMemberActivity, Message: "Miles in AK"},
			{Code: ifta.ProblemNoFuelForType, Message: "No CNG purchases"},
		}

		assert.True(t, r.CanFinalize())
		assert.False(t, r.HasProblem(ifta.ProblemMissingRate))
		assert.Empty(t, r.BlockingProblems())
	})

	t.Run("only MissingRate blocks", func(t *testing.T) {
		t.Parallel()

		for _, code := range []ifta.ProblemCode{
			ifta.ProblemNoFuelForType,
			ifta.ProblemUnattributedMiles,
			ifta.ProblemNoTractorMiles,
			ifta.ProblemMileageMismatch,
			ifta.ProblemNonMemberActivity,
			ifta.ProblemNonQualifiedActivity,
		} {
			assert.False(t, code.Blocks(), "%s must not block", code)
			assert.True(t, code.IsValid())
			assert.NotEmpty(t, code.Label())
		}
		assert.True(t, ifta.ProblemMissingRate.Blocks())
		assert.False(t, ifta.ProblemCode("Unknown").IsValid())
	})
}

func TestReturn_ValidStatesPass(t *testing.T) {
	t.Parallel()

	for name, r := range map[string]*ifta.Return{
		"draft":     draftReturn(),
		"finalized": finalizedReturn(),
		"filed":     filedReturn(),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r.Normalize()

			multiErr := errortypes.NewMultiError()
			r.Validate(multiErr)

			assert.False(t, multiErr.HasErrors(), multiErr.Error())
		})
	}
}

func TestReturn_ValidateRejections(t *testing.T) {
	t.Parallel()

	parent := pulid.MustNew("ifr_")

	tests := []struct {
		name   string
		build  func() *ifta.Return
		mutate func(r *ifta.Return)
		field  string
	}{
		{
			name:   "reopen reason too short",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.ReopenedAt = ptrInt64(1); r.ReopenReason = "typo" },
			field:  "reopenReason",
		},
		{
			name:   "reopen stamp without a reason",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.ReopenedAt = ptrInt64(1) },
			field:  "reopenReason",
		},
		{
			name:   "finalized without a stamp",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.Status = ifta.ReturnStatusFinalized },
			field:  "finalizedAt",
		},
		{
			name:   "draft with a finalized stamp",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.FinalizedAt = ptrInt64(1) },
			field:  "finalizedAt",
		},
		{
			name:   "filed without a filed stamp",
			build:  finalizedReturn,
			mutate: func(r *ifta.Return) { r.Status = ifta.ReturnStatusFiled },
			field:  "filedAt",
		},
		{
			name:   "finalized with a filed stamp",
			build:  finalizedReturn,
			mutate: func(r *ifta.Return) { r.FiledAt = ptrInt64(1_783_100_000) },
			field:  "filedAt",
		},
		{
			name:   "filed before finalized",
			build:  filedReturn,
			mutate: func(r *ifta.Return) { r.FiledAt = ptrInt64(*r.FinalizedAt - 1) },
			field:  "filedAt",
		},
		{
			name:   "amendment without a parent",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.AmendmentNumber = 1 },
			field:  "amendsReturnId",
		},
		{
			name:   "original with a parent",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.AmendsReturnID = &parent },
			field:  "amendsReturnId",
		},
		{
			name:   "negative amendment number",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.AmendmentNumber = -1 },
			field:  "amendmentNumber",
		},
		{
			name:   "period end before start",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.PeriodEnd = r.PeriodStart },
			field:  "periodEnd",
		},
		{
			name:   "quarter out of range",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.Quarter = 5 },
			field:  "quarter",
		},
		{
			name:   "missing timezone",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.Timezone = "" },
			field:  "timezone",
		},
		{
			name:   "bad currency",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.CurrencyCode = "usd" },
			field:  "currencyCode",
		},
		{
			name:   "bad status",
			build:  draftReturn,
			mutate: func(r *ifta.Return) { r.Status = "Submitted" },
			field:  "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := tt.build()
			tt.mutate(r)

			multiErr := errortypes.NewMultiError()
			r.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}

func TestReturn_ReopenReasonAtMinimumLengthPasses(t *testing.T) {
	t.Parallel()

	r := draftReturn()
	r.ReopenedAt = ptrInt64(1_783_200_000)
	r.ReopenedByID = pulid.MustNew("usr_")
	r.ReopenReason = "0123456789"
	require.Len(t, r.ReopenReason, ifta.MinReasonLength)

	multiErr := errortypes.NewMultiError()
	r.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestReturn_AmendmentPasses(t *testing.T) {
	t.Parallel()

	parent := pulid.MustNew("ifr_")
	r := draftReturn()
	r.AmendmentNumber = 1
	r.AmendsReturnID = &parent

	multiErr := errortypes.NewMultiError()
	r.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, r.IsAmendment())
}

func TestReturn_MoneyAndPeriod(t *testing.T) {
	t.Parallel()

	r := draftReturn()
	r.TaxDueMinor = -12_345
	r.SurchargeDueMinor = 2_000
	r.NetDueMinor = -10_345

	assert.Equal(t, "-123.45", r.TaxDue().StringFixed(2))
	assert.Equal(t, "20.00", r.SurchargeDue().StringFixed(2))
	assert.Equal(t, "-103.45", r.NetDue().StringFixed(2))
	assert.True(t, r.IsCredit())
	assert.Equal(t, ifta.NewPeriod(2026, 2), r.Period())
	assert.Equal(t, "ifta_returns", r.GetTableName())
	assert.Equal(t, "ifta_return", r.GetResourceType())
}

func TestReturnLine_Validate(t *testing.T) {
	t.Parallel()

	validLine := func() *ifta.ReturnLine {
		return &ifta.ReturnLine{
			ReturnID:       pulid.MustNew("ifr_"),
			JurisdictionID: pulid.MustNew("ifj_"),
			FuelType:       domaintypes.IFTAFuelTypeDiesel,
			IsIftaMember:   true,
			RateMissing:    true,
			TaxDueMinor:    -500,
			LineTotalMinor: -500,
		}
	}

	t.Run("missing-rate line without a rate passes", func(t *testing.T) {
		t.Parallel()

		line := validLine()
		multiErr := errortypes.NewMultiError()
		line.Validate(multiErr)

		assert.False(t, multiErr.HasErrors(), multiErr.Error())
		assert.True(t, line.IsCredit())
		assert.Equal(t, "-5.00", line.LineTotal().StringFixed(2))
	})

	tests := []struct {
		name   string
		mutate func(l *ifta.ReturnLine)
		field  string
	}{
		{
			name:   "rate present flag without a rate",
			mutate: func(l *ifta.ReturnLine) { l.RateMissing = false },
			field:  "ratePerGallon",
		},
		{
			name:   "negative surcharge",
			mutate: func(l *ifta.ReturnLine) { l.SurchargeDueMinor = -1; l.LineTotalMinor = -501 },
			field:  "surchargeDueMinor",
		},
		{
			name:   "line total drift",
			mutate: func(l *ifta.ReturnLine) { l.LineTotalMinor = 0 },
			field:  "lineTotalMinor",
		},
		{
			name:   "DEF is not an IFTA line",
			mutate: func(l *ifta.ReturnLine) { l.FuelType = "Kerosene" },
			field:  "fuelType",
		},
		{
			name:   "negative purchase count",
			mutate: func(l *ifta.ReturnLine) { l.PurchaseCount = -1 },
			field:  "purchaseCount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			line := validLine()
			tt.mutate(line)

			multiErr := errortypes.NewMultiError()
			line.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}
