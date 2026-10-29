package factoring

import "github.com/emoss08/trenova/shared/timeutils"

type Funding struct {
	FaceMinor          int64 `json:"faceMinor"`
	AdvanceMinor       int64 `json:"advanceMinor"`
	FeeMinor           int64 `json:"feeMinor"`
	ReserveMinor       int64 `json:"reserveMinor"`
	RestOnReleaseMinor int64 `json:"restOnReleaseMinor"`
	Days               int   `json:"days"`
	FeeRateBP          int   `json:"feeRateBp"`
}

func ExpectedFunding(faceMinor int64, agreement *Agreement, days int) Funding {
	if days < 0 {
		days = 0
	}
	out := Funding{FaceMinor: faceMinor, Days: days}
	if agreement == nil || faceMinor <= 0 {
		return out
	}
	out.ReserveMinor = basisPointsOf(faceMinor, agreement.ReserveRateBP)
	out.FeeRateBP, out.FeeMinor = feeFor(faceMinor, agreement, days)
	out.AdvanceMinor = advanceLeavingRoomForFee(faceMinor, agreement.AdvanceRateBP, out.FeeMinor)
	out.RestOnReleaseMinor = faceMinor - out.AdvanceMinor - out.FeeMinor
	return out
}

func feeFor(faceMinor int64, agreement *Agreement, days int) (int, int64) {
	var rateBP int
	var fee int64
	if tier := tierFor(agreement.FeeTiers, days); tier != nil {
		rateBP = tier.RateBP
		fee = basisPointsOf(faceMinor, tier.RateBP) + tier.FlatMinor
	}
	return rateBP, min(max(fee, agreement.MinimumFeeMinor), faceMinor)
}

func advanceLeavingRoomForFee(faceMinor int64, advanceRateBP int, feeMinor int64) int64 {
	return min(basisPointsOf(faceMinor, advanceRateBP), faceMinor-feeMinor)
}

func DaysToRun(dueDate *int64, at int64) int {
	if dueDate == nil || *dueDate <= at {
		return 0
	}
	return int((*dueDate - at + timeutils.SecondsPerDay - 1) / timeutils.SecondsPerDay)
}

func tierFor(tiers []*FeeTier, days int) *FeeTier {
	ordered := SortedFeeTiers(tiers)
	var last *FeeTier
	for _, tier := range ordered {
		if tier == nil {
			continue
		}
		last = tier
		if days >= tier.DaysFrom && (tier.DaysTo == nil || days < *tier.DaysTo) {
			return tier
		}
	}
	return last
}

func basisPointsOf(amount int64, bp int) int64 {
	if amount <= 0 || bp <= 0 {
		return 0
	}
	return (amount*int64(bp) + BasisPoints/2) / BasisPoints
}
