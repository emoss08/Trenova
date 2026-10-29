package factoring_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/stretchr/testify/assert"
)

func TestExpectedFunding(t *testing.T) {
	t.Parallel()

	agreement := validAgreement()
	agreement.MinimumFeeMinor = 1500

	tests := []struct {
		name string
		face int64
		days int
		want factoring.Funding
	}{
		{
			name: "first tier",
			face: 100000,
			days: 10,
			want: factoring.Funding{
				FaceMinor: 100000, AdvanceMinor: 90000, FeeMinor: 3000,
				ReserveMinor: 10000, RestOnReleaseMinor: 7000, Days: 10, FeeRateBP: 300,
			},
		},
		{
			name: "open-ended tier",
			face: 100000,
			days: 45,
			want: factoring.Funding{
				FaceMinor: 100000, AdvanceMinor: 90000, FeeMinor: 5000,
				ReserveMinor: 10000, RestOnReleaseMinor: 5000, Days: 45, FeeRateBP: 500,
			},
		},
		{
			name: "minimum fee",
			face: 10000,
			days: 0,
			want: factoring.Funding{
				FaceMinor: 10000, AdvanceMinor: 8500, FeeMinor: 1500,
				ReserveMinor: 1000, RestOnReleaseMinor: 0, Days: 0, FeeRateBP: 300,
			},
		},
		{
			name: "rounds half up",
			face: 15,
			days: -3,
			want: factoring.Funding{
				FaceMinor: 15, AdvanceMinor: 0, FeeMinor: 15,
				ReserveMinor: 2, RestOnReleaseMinor: 0, Days: 0, FeeRateBP: 300,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, factoring.ExpectedFunding(tt.face, agreement, tt.days))
		})
	}
}

func TestExpectedFundingAtTheRateLimit(t *testing.T) {
	t.Parallel()

	agreement := validAgreement()
	agreement.AdvanceRateBP = 9400
	agreement.ReserveRateBP = 300
	agreement.FeeTiers = []*factoring.FeeTier{
		{DaysFrom: 0, DaysTo: intPtr(75), RateBP: 425},
		{DaysFrom: 75, RateBP: 600},
	}
	assert.Equal(t, factoring.Funding{
		FaceMinor: 200000, AdvanceMinor: 188000, FeeMinor: 12000,
		ReserveMinor: 6000, RestOnReleaseMinor: 0, Days: 88, FeeRateBP: 600,
	}, factoring.ExpectedFunding(200000, agreement, 88))
	assert.Equal(t, factoring.Funding{
		FaceMinor: 25, AdvanceMinor: 23, FeeMinor: 2,
		ReserveMinor: 1, RestOnReleaseMinor: 0, Days: 88, FeeRateBP: 600,
	}, factoring.ExpectedFunding(25, agreement, 88), "both rates round up; the advance gives way")
}

func TestExpectedFundingFlatFeeOnASmallInvoice(t *testing.T) {
	t.Parallel()

	agreement := validAgreement()
	agreement.FeeTiers = []*factoring.FeeTier{{DaysFrom: 0, RateBP: 300, FlatMinor: 2500}}
	assert.Equal(t, factoring.Funding{
		FaceMinor: 20000, AdvanceMinor: 16900, FeeMinor: 3100,
		ReserveMinor: 2000, RestOnReleaseMinor: 0, Days: 5, FeeRateBP: 300,
	}, factoring.ExpectedFunding(20000, agreement, 5))
}

func TestExpectedFundingWithoutAgreement(t *testing.T) {
	t.Parallel()

	got := factoring.ExpectedFunding(5000, nil, 3)
	assert.Equal(t, factoring.Funding{FaceMinor: 5000, Days: 3}, got)
}

func TestExpectedFundingPastClosedTiers(t *testing.T) {
	t.Parallel()

	agreement := validAgreement()
	agreement.FeeTiers = []*factoring.FeeTier{{DaysFrom: 0, DaysTo: intPtr(30), RateBP: 250}}
	got := factoring.ExpectedFunding(100000, agreement, 90)
	assert.Equal(t, int64(2500), got.FeeMinor)
}

func TestDaysToRun(t *testing.T) {
	t.Parallel()

	due := int64(now + 86400*10 + 1)
	past := int64(now - 86400)
	assert.Equal(t, 11, factoring.DaysToRun(&due, now))
	assert.Equal(t, 0, factoring.DaysToRun(&past, now))
	assert.Equal(t, 0, factoring.DaysToRun(nil, now))
}
