package factoringoverviewservice

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	MaxPreviewFaceMinor = int64(100_000_000_000)
	MaxPreviewDays      = 3650
)

type FundingPreviewRequest struct {
	FaceMinor       int64
	Days            int
	AdvanceRateBP   int
	ReserveRateBP   int
	MinimumFeeMinor int64
	FeeTiers        []*factoring.FeeTier
}

func (s *Service) PreviewFunding(req *FundingPreviewRequest) (factoring.Funding, error) {
	multiErr := errortypes.NewMultiError()
	if req.FaceMinor < 0 || req.FaceMinor > MaxPreviewFaceMinor {
		multiErr.Add("faceMinor", errortypes.ErrInvalid, "Enter an invoice amount up to one billion")
	}
	if req.Days < 0 || req.Days > MaxPreviewDays {
		multiErr.Add("days", errortypes.ErrInvalid, "Days left must be between 0 and 3,650")
	}
	if req.AdvanceRateBP < 0 || req.AdvanceRateBP > factoring.BasisPoints ||
		req.ReserveRateBP < 0 || req.ReserveRateBP > factoring.BasisPoints {
		multiErr.Add("advanceRateBp", errortypes.ErrInvalid, "Rates must be between 0% and 100%")
	}
	if req.MinimumFeeMinor < 0 || req.MinimumFeeMinor > MaxPreviewFaceMinor {
		multiErr.Add("minimumFeeMinor", errortypes.ErrInvalid, "The minimum fee cannot be negative")
	}
	if len(req.FeeTiers) > factoring.MaxFeeTiers {
		multiErr.Add("feeTiers", errortypes.ErrInvalid, "An agreement can have at most 20 fee tiers")
	}
	for _, tier := range req.FeeTiers {
		if tier.DaysFrom < 0 || tier.RateBP < 0 || tier.RateBP > factoring.BasisPoints ||
			tier.FlatMinor < 0 || tier.FlatMinor > MaxPreviewFaceMinor ||
			(tier.DaysTo != nil && *tier.DaysTo <= tier.DaysFrom) {
			multiErr.Add("feeTiers", errortypes.ErrInvalid, "Each fee tier needs a valid range and rate")
			break
		}
	}
	if multiErr.HasErrors() {
		return factoring.Funding{}, multiErr
	}

	return factoring.ExpectedFunding(req.FaceMinor, &factoring.Agreement{
		AdvanceRateBP:   req.AdvanceRateBP,
		ReserveRateBP:   req.ReserveRateBP,
		MinimumFeeMinor: req.MinimumFeeMinor,
		FeeTiers:        req.FeeTiers,
	}, req.Days), nil
}
