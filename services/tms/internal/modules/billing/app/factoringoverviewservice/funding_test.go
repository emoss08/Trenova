package factoringoverviewservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func previewRequest() *FundingPreviewRequest {
	return &FundingPreviewRequest{
		FaceMinor:       1_000_000,
		Days:            30,
		AdvanceRateBP:   9000,
		ReserveRateBP:   700,
		MinimumFeeMinor: 1500,
		FeeTiers: []*factoring.FeeTier{
			{DaysFrom: 30, RateBP: 250},
			{DaysFrom: 0, DaysTo: new(30), RateBP: 200},
		},
	}
}

func TestPreviewFundingPricesTheInvoiceAsFundingDoes(t *testing.T) {
	t.Parallel()

	req := previewRequest()
	got, err := newService(&fakeRepo{}).PreviewFunding(req)
	require.NoError(t, err)

	want := factoring.ExpectedFunding(req.FaceMinor, &factoring.Agreement{
		AdvanceRateBP:   req.AdvanceRateBP,
		ReserveRateBP:   req.ReserveRateBP,
		MinimumFeeMinor: req.MinimumFeeMinor,
		FeeTiers:        req.FeeTiers,
	}, req.Days)
	assert.Equal(t, want, got)
	assert.Equal(t, factoring.Funding{
		FaceMinor: 1_000_000, AdvanceMinor: 900_000, FeeMinor: 25_000, ReserveMinor: 70_000,
		RestOnReleaseMinor: 75_000, Days: 30, FeeRateBP: 250,
	}, got)
}

func TestPreviewFundingRefusesTermsOutsideTheirBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		edit  func(*FundingPreviewRequest)
		field string
	}{
		{"a negative amount", func(r *FundingPreviewRequest) { r.FaceMinor = -1 }, "faceMinor"},
		{
			"more days than ten years",
			func(r *FundingPreviewRequest) { r.Days = MaxPreviewDays + 1 },
			"days",
		},
		{
			"an advance over 100%",
			func(r *FundingPreviewRequest) { r.AdvanceRateBP = 10_001 },
			"advanceRateBp",
		},
		{
			"a negative minimum fee",
			func(r *FundingPreviewRequest) { r.MinimumFeeMinor = -5 },
			"minimumFeeMinor",
		},
		{
			"a tier that ends before it starts",
			func(r *FundingPreviewRequest) { r.FeeTiers[1].DaysTo = new(0) },
			"feeTiers",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := previewRequest()
			tt.edit(req)
			_, err := newService(&fakeRepo{}).PreviewFunding(req)
			var multi *errortypes.MultiError
			require.ErrorAs(t, err, &multi)
			require.NotEmpty(t, multi.Errors)
			assert.Equal(t, tt.field, multi.Errors[0].Field)
		})
	}
}
