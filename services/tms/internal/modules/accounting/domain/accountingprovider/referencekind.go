package accountingprovider

type ReferenceKind string

const (
	ReferenceKindAccount       = ReferenceKind("Account")
	ReferenceKindItem          = ReferenceKind("Item")
	ReferenceKindCustomer      = ReferenceKind("Customer")
	ReferenceKindVendor        = ReferenceKind("Vendor")
	ReferenceKindTerm          = ReferenceKind("Term")
	ReferenceKindPaymentMethod = ReferenceKind("PaymentMethod")
)

func (k ReferenceKind) String() string { return string(k) }

func (k ReferenceKind) IsValid() bool {
	switch k {
	case ReferenceKindAccount,
		ReferenceKindItem,
		ReferenceKindCustomer,
		ReferenceKindVendor,
		ReferenceKindTerm,
		ReferenceKindPaymentMethod:
		return true
	default:
		return false
	}
}

func (k ReferenceKind) Creatable() bool {
	return k == ReferenceKindItem || k == ReferenceKindCustomer || k == ReferenceKindVendor
}

func AllReferenceKinds() []ReferenceKind {
	return []ReferenceKind{
		ReferenceKindAccount,
		ReferenceKindItem,
		ReferenceKindCustomer,
		ReferenceKindVendor,
		ReferenceKindTerm,
		ReferenceKindPaymentMethod,
	}
}
