package shipmentvocab

type ChargeAllocationKind string

const (
	ChargeAllocationKindFreight     = ChargeAllocationKind("Freight")
	ChargeAllocationKindAccessorial = ChargeAllocationKind("Accessorial")
	ChargeAllocationKindOrderCharge = ChargeAllocationKind("OrderCharge")
)

func (k ChargeAllocationKind) IsValid() bool {
	switch k {
	case ChargeAllocationKindFreight,
		ChargeAllocationKindAccessorial,
		ChargeAllocationKindOrderCharge:
		return true
	default:
		return false
	}
}

type ChargeAllocationMethod string

const (
	ChargeAllocationMethodPercent = ChargeAllocationMethod("Percent")
	ChargeAllocationMethodAmount  = ChargeAllocationMethod("Amount")
)

func ChargeAllocationMethodValues() []ChargeAllocationMethod {
	return []ChargeAllocationMethod{ChargeAllocationMethodPercent, ChargeAllocationMethodAmount}
}

func (m ChargeAllocationMethod) IsValid() bool {
	switch m {
	case ChargeAllocationMethodPercent, ChargeAllocationMethodAmount:
		return true
	default:
		return false
	}
}
