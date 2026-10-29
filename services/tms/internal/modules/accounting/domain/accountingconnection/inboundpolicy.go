package accountingconnection

type InboundPaymentPolicy string

const (
	InboundPaymentsOff     = InboundPaymentPolicy("Off")
	InboundPaymentsPropose = InboundPaymentPolicy("Propose")
	InboundPaymentsApply   = InboundPaymentPolicy("Apply")
)

func (p InboundPaymentPolicy) String() string { return string(p) }

func (p InboundPaymentPolicy) IsValid() bool {
	switch p {
	case InboundPaymentsOff, InboundPaymentsPropose, InboundPaymentsApply:
		return true
	default:
		return false
	}
}

func AllInboundPaymentPolicies() []InboundPaymentPolicy {
	return []InboundPaymentPolicy{InboundPaymentsOff, InboundPaymentsPropose, InboundPaymentsApply}
}
