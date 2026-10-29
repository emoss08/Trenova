package accountingsync

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
)

type MappingTargetType string

const (
	TargetAccountRole       = MappingTargetType("AccountRole")
	TargetLineType          = MappingTargetType("LineType")
	TargetAccessorialCharge = MappingTargetType("AccessorialCharge")
	TargetItemRole          = MappingTargetType("ItemRole")
	TargetCustomer          = MappingTargetType("Customer")
	TargetCarrier           = MappingTargetType("Carrier")
	TargetDriver            = MappingTargetType("Driver")
	TargetGLAccount         = MappingTargetType("GLAccount")
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
		TargetDriver,
		TargetGLAccount,
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
		TargetDriver,
		TargetGLAccount,
		TargetPaymentTerm,
		TargetPaymentMethod,
	}
}

func (t MappingTargetType) KeyedByObject() bool {
	switch t {
	case TargetAccessorialCharge, TargetCustomer, TargetCarrier, TargetDriver, TargetGLAccount:
		return true
	case TargetAccountRole, TargetLineType, TargetItemRole, TargetPaymentTerm, TargetPaymentMethod:
		return false
	default:
		return false
	}
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
	case TargetAccessorialCharge, TargetCustomer, TargetCarrier, TargetDriver, TargetGLAccount:
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
