package factoringledger

type FundingOutcome string

const (
	FundingOutcomeFunded   = FundingOutcome("Funded")
	FundingOutcomeRejected = FundingOutcome("Rejected")
)

func (o FundingOutcome) IsValid() bool {
	return o == FundingOutcomeFunded || o == FundingOutcomeRejected
}

func AllFundingOutcomes() []FundingOutcome {
	return []FundingOutcome{FundingOutcomeFunded, FundingOutcomeRejected}
}

type ReserveEntryKind string

const (
	ReserveEntryFunding            = ReserveEntryKind("Funding")
	ReserveEntryRelease            = ReserveEntryKind("ReserveRelease")
	ReserveEntryChargeback         = ReserveEntryKind("Chargeback")
	ReserveEntryChargebackReversal = ReserveEntryKind("ChargebackReversal")
)

func (k ReserveEntryKind) IsValid() bool {
	switch k {
	case ReserveEntryFunding, ReserveEntryRelease, ReserveEntryChargeback,
		ReserveEntryChargebackReversal:
		return true
	default:
		return false
	}
}
