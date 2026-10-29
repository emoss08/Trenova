package payescrow

type EscrowAccountStatus string

const (
	EscrowAccountStatusActive = EscrowAccountStatus("Active")
	EscrowAccountStatusClosed = EscrowAccountStatus("Closed")
)

func (e EscrowAccountStatus) String() string { return string(e) }

func (e EscrowAccountStatus) IsValid() bool {
	switch e {
	case EscrowAccountStatusActive, EscrowAccountStatusClosed:
		return true
	default:
		return false
	}
}

type EscrowTransactionType string

const (
	EscrowTransactionTypeContribution    = EscrowTransactionType("Contribution")
	EscrowTransactionTypeInterestAccrual = EscrowTransactionType("InterestAccrual")
	EscrowTransactionTypeApplication     = EscrowTransactionType("Application")
	EscrowTransactionTypeRefund          = EscrowTransactionType("Refund")
	EscrowTransactionTypeAdjustment      = EscrowTransactionType("Adjustment")
)

func (e EscrowTransactionType) String() string { return string(e) }

func (e EscrowTransactionType) IsValid() bool {
	switch e {
	case EscrowTransactionTypeContribution, EscrowTransactionTypeInterestAccrual,
		EscrowTransactionTypeApplication, EscrowTransactionTypeRefund,
		EscrowTransactionTypeAdjustment:
		return true
	default:
		return false
	}
}
