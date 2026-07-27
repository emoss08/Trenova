package detention_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tier(from int32, to *int32, rate string) *detention.DetentionPolicyTier {
	return &detention.DetentionPolicyTier{
		FromMinute: from,
		ToMinute:   to,
		Rate:       decimal.RequireFromString(rate),
		RateUnit:   detention.TierRateUnitHour,
	}
}

func tieredPolicy(tiers ...*detention.DetentionPolicyTier) *detention.DetentionPolicy {
	p := validPolicy()
	p.RateSource = detention.RateSourceTiers
	p.Tiers = tiers
	return p
}

func TestTier_SpanMinutes(t *testing.T) {
	t.Parallel()

	first := tier(0, ptr(int32(120)), "75")
	second := tier(120, nil, "100")

	tests := []struct {
		name          string
		tier          *detention.DetentionPolicyTier
		totalBillable int32
		want          int32
	}{
		{"bounded tier fully consumed", first, 300, 120},
		{"bounded tier partially consumed", first, 45, 45},
		{"bounded tier untouched", first, 0, 0},
		{"open tier not yet reached", second, 120, 0},
		{"open tier absorbs the remainder", second, 300, 180},
		{"total below the tier floor", second, 60, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.tier.SpanMinutes(tt.totalBillable))
		})
	}
}

func TestTier_Covers(t *testing.T) {
	t.Parallel()

	band := tier(60, ptr(int32(120)), "75")

	assert.False(t, band.Covers(59))
	assert.True(t, band.Covers(60), "the lower bound is inclusive")
	assert.True(t, band.Covers(119))
	assert.False(t, band.Covers(120), "the upper bound is exclusive")
}

func TestTier_SortTiers(t *testing.T) {
	t.Parallel()

	sorted := detention.SortTiers([]*detention.DetentionPolicyTier{
		tier(240, nil, "125"),
		nil,
		tier(0, ptr(int32(120)), "75"),
		tier(120, ptr(int32(240)), "100"),
	})

	require.Len(t, sorted, 3, "nil entries are dropped")
	assert.Equal(t, int32(0), sorted[0].FromMinute)
	assert.Equal(t, int32(120), sorted[1].FromMinute)
	assert.Equal(t, int32(240), sorted[2].FromMinute)
}

func TestTier_ValidationAcceptsContiguousLadder(t *testing.T) {
	t.Parallel()

	p := tieredPolicy(
		tier(0, ptr(int32(120)), "75"),
		tier(120, ptr(int32(480)), "100"),
		tier(480, nil, "125"),
	)

	assert.False(t, validate(p).HasErrors())
}

func TestTier_ValidationRejectsBadLadders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tiers     []*detention.DetentionPolicyTier
		wantField string
	}{
		{
			name:      "ladder does not start at zero",
			tiers:     []*detention.DetentionPolicyTier{tier(60, nil, "75")},
			wantField: "tiers[0].fromMinute",
		},
		{
			name: "overlapping bands",
			tiers: []*detention.DetentionPolicyTier{
				tier(0, ptr(int32(120)), "75"),
				tier(90, nil, "100"),
			},
			wantField: "tiers[1].fromMinute",
		},
		{
			name: "gap between bands leaves unbilled time",
			tiers: []*detention.DetentionPolicyTier{
				tier(0, ptr(int32(120)), "75"),
				tier(180, nil, "100"),
			},
			wantField: "tiers[1].fromMinute",
		},
		{
			name: "band ends before it starts",
			tiers: []*detention.DetentionPolicyTier{
				tier(0, ptr(int32(120)), "75"),
				tier(120, ptr(int32(60)), "100"),
			},
			wantField: "tiers[1].toMinute",
		},
		{
			name: "negative tier start",
			tiers: []*detention.DetentionPolicyTier{
				tier(-10, ptr(int32(120)), "75"),
			},
			wantField: "tiers[0].fromMinute",
		},
		{
			name: "negative rate",
			tiers: []*detention.DetentionPolicyTier{
				tier(0, nil, "-5"),
			},
			wantField: "tiers[0].rate",
		},
		{
			name: "a tier after an open-ended tier",
			tiers: []*detention.DetentionPolicyTier{
				tier(0, nil, "75"),
				tier(120, ptr(int32(240)), "100"),
			},
			wantField: "tiers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := validate(tieredPolicy(tt.tiers...))

			require.True(t, multiErr.HasErrors())
			assert.Contains(t, errorFields(multiErr), tt.wantField)
		})
	}
}

func TestTier_ValidationRejectsInvalidRateUnit(t *testing.T) {
	t.Parallel()

	bad := tier(0, nil, "75")
	bad.RateUnit = detention.TierRateUnit("Fortnight")

	multiErr := validate(tieredPolicy(bad))

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, errorFields(multiErr), "tiers[0].rateUnit")
}

func TestTier_MultipleOpenEndedTiersRejected(t *testing.T) {
	t.Parallel()

	multiErr := validate(tieredPolicy(
		tier(0, nil, "75"),
		tier(0, nil, "100"),
	))

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, errorFields(multiErr), "tiers")
}

func TestTier_SnapshotRoundTripsLadder(t *testing.T) {
	t.Parallel()

	p := tieredPolicy(
		tier(120, ptr(int32(480)), "100"),
		tier(0, ptr(int32(120)), "75"),
	)
	p.ID = pulid.MustNew("dtp_")

	snapshot := detention.NewPolicySnapshot(detention.SnapshotInput{
		Policy:      p,
		FreeMinutes: 120,
		CapturedAt:  1000,
	})

	require.Len(t, snapshot.Tiers, 2)
	assert.Equal(t, int32(0), snapshot.Tiers[0].FromMinute,
		"the snapshot stores the ladder already sorted")
	assert.Equal(t, int32(120), snapshot.Tiers[1].FromMinute)
	assert.Equal(t, int64(1000), snapshot.CapturedAt)
	assert.Equal(t, p.ID, snapshot.PolicyID)
}

func TestSnapshot_OmitsTiersForFlatRatePolicies(t *testing.T) {
	t.Parallel()

	p := validPolicy()
	p.RateSource = detention.RateSourceAccessorial
	p.Tiers = []*detention.DetentionPolicyTier{tier(0, nil, "75")}

	snapshot := detention.NewPolicySnapshot(detention.SnapshotInput{
		Policy:      p,
		FreeMinutes: 120,
		CapturedAt:  1000,
	})

	assert.Empty(t, snapshot.Tiers,
		"a flat-rate policy must not carry a stale ladder into the snapshot")
}

func TestSnapshot_NilPolicyIsSafe(t *testing.T) {
	t.Parallel()

	snapshot := detention.NewPolicySnapshot(detention.SnapshotInput{CapturedAt: 42})

	assert.Equal(t, int64(42), snapshot.CapturedAt)
	assert.True(t, snapshot.PolicyID.IsNil())
}
