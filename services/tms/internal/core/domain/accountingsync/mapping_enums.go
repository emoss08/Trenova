package accountingsync

import (
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
)

type SetupStep string

const (
	SetupStepMappings = SetupStep("Mappings")
	SetupStepComplete = SetupStep("Complete")
)

func (s SetupStep) String() string { return string(s) }

func (s SetupStep) IsValid() bool {
	switch s {
	case SetupStepMappings, SetupStepComplete:
		return true
	default:
		return false
	}
}

func AllSetupSteps() []SetupStep {
	return []SetupStep{SetupStepMappings, SetupStepComplete}
}

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

type MappingTargetType string

const (
	TargetAccountRole       = MappingTargetType("AccountRole")
	TargetLineType          = MappingTargetType("LineType")
	TargetAccessorialCharge = MappingTargetType("AccessorialCharge")
	TargetItemRole          = MappingTargetType("ItemRole")
	TargetCustomer          = MappingTargetType("Customer")
	TargetCarrier           = MappingTargetType("Carrier")
	TargetPaymentTerm       = MappingTargetType("PaymentTerm")
	TargetPaymentMethod     = MappingTargetType("PaymentMethod")
)

func (t MappingTargetType) String() string { return string(t) }

func (t MappingTargetType) IsValid() bool {
	switch t {
	case TargetAccountRole,
		TargetLineType,
		TargetAccessorialCharge,
		TargetItemRole,
		TargetCustomer,
		TargetCarrier,
		TargetPaymentTerm,
		TargetPaymentMethod:
		return true
	default:
		return false
	}
}

func AllMappingTargetTypes() []MappingTargetType {
	return []MappingTargetType{
		TargetAccountRole,
		TargetLineType,
		TargetAccessorialCharge,
		TargetItemRole,
		TargetCustomer,
		TargetCarrier,
		TargetPaymentTerm,
		TargetPaymentMethod,
	}
}

func (t MappingTargetType) ProviderKind() ReferenceKind {
	switch t {
	case TargetAccountRole:
		return ReferenceKindAccount
	case TargetLineType, TargetAccessorialCharge, TargetItemRole:
		return ReferenceKindItem
	case TargetCustomer:
		return ReferenceKindCustomer
	case TargetCarrier:
		return ReferenceKindVendor
	case TargetPaymentTerm:
		return ReferenceKindTerm
	case TargetPaymentMethod:
		return ReferenceKindPaymentMethod
	default:
		return ""
	}
}

func (t MappingTargetType) KeyedByObject() bool {
	return t == TargetAccessorialCharge || t == TargetCustomer || t == TargetCarrier
}

func (t MappingTargetType) Keys() []string {
	switch t {
	case TargetAccountRole:
		return []string{
			AccountRoleAR,
			AccountRoleRevenue,
			AccountRoleDeposit,
			AccountRoleWriteOff,
			AccountRoleAP,
			AccountRolePurchasedTransportation,
		}
	case TargetLineType:
		return []string{string(invoice.InvoiceLineTypeFreight), string(invoice.InvoiceLineTypeMemo)}
	case TargetItemRole:
		return []string{ItemRoleShortPayWriteOff}
	case TargetPaymentTerm:
		return []string{
			string(customer.PaymentTermNet10),
			string(customer.PaymentTermNet15),
			string(customer.PaymentTermNet30),
			string(customer.PaymentTermNet45),
			string(customer.PaymentTermNet60),
			string(customer.PaymentTermNet90),
			string(customer.PaymentTermDueOnReceipt),
		}
	case TargetPaymentMethod:
		return []string{
			string(customerpayment.MethodACH),
			string(customerpayment.MethodCheck),
			string(customerpayment.MethodWire),
			string(customerpayment.MethodCard),
			string(customerpayment.MethodCash),
			string(customerpayment.MethodOther),
		}
	case TargetAccessorialCharge, TargetCustomer, TargetCarrier:
		return nil
	default:
		return nil
	}
}

func (t MappingTargetType) AcceptsKey(key string) bool {
	for _, candidate := range t.Keys() {
		if candidate == key {
			return true
		}
	}
	return false
}

const (
	AccountRoleAR                      = "ARAccount"
	AccountRoleRevenue                 = "RevenueAccount"
	AccountRoleDeposit                 = "DepositAccount"
	AccountRoleWriteOff                = "WriteOffAccount"
	AccountRoleAP                      = "APAccount"
	AccountRolePurchasedTransportation = "PurchasedTransportationAccount"

	ItemRoleShortPayWriteOff = "ShortPayWriteOff"
)

type MappingState string

const (
	MappingStateUnmatched = MappingState("Unmatched")
	MappingStateProposed  = MappingState("Proposed")
	MappingStateConfirmed = MappingState("Confirmed")
)

func (s MappingState) String() string { return string(s) }

func (s MappingState) IsValid() bool {
	switch s {
	case MappingStateUnmatched, MappingStateProposed, MappingStateConfirmed:
		return true
	default:
		return false
	}
}

func AllMappingStates() []MappingState {
	return []MappingState{MappingStateUnmatched, MappingStateProposed, MappingStateConfirmed}
}

type MappingSource string

const (
	MappingSourceSuggested         = MappingSource("Suggested")
	MappingSourceModel             = MappingSource("Model")
	MappingSourceManual            = MappingSource("Manual")
	MappingSourceCreatedInProvider = MappingSource("CreatedInProvider")
	MappingSourceAgent             = MappingSource("Agent")
)

func (s MappingSource) String() string { return string(s) }

func (s MappingSource) IsValid() bool {
	switch s {
	case MappingSourceSuggested,
		MappingSourceModel,
		MappingSourceManual,
		MappingSourceCreatedInProvider,
		MappingSourceAgent:
		return true
	default:
		return false
	}
}

func AllMappingSources() []MappingSource {
	return []MappingSource{
		MappingSourceSuggested,
		MappingSourceModel,
		MappingSourceManual,
		MappingSourceCreatedInProvider,
		MappingSourceAgent,
	}
}

func (s MappingSource) Deliberate() bool {
	return s == MappingSourceManual || s == MappingSourceCreatedInProvider ||
		s == MappingSourceAgent
}
