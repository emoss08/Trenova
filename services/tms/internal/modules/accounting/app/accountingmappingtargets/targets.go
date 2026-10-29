package accountingmappingtargets

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingscore"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const targetPageSize = pagination.MaxLimit

var roleLabels = map[string]string{
	accountingsync.AccountRoleAR:                      "Accounts receivable",
	accountingsync.AccountRoleRevenue:                 "Revenue",
	accountingsync.AccountRoleDeposit:                 "Deposit account",
	accountingsync.AccountRoleWriteOff:                "Short pay and write-off",
	accountingsync.AccountRoleAP:                      "Accounts payable",
	accountingsync.AccountRolePurchasedTransportation: "Purchased transportation",
}

var lineTypeLabels = map[string]string{
	string(invoice.InvoiceLineTypeFreight): "Freight charges",
	string(invoice.InvoiceLineTypeMemo):    "Memo lines",
}

var itemRoleLabels = map[string]string{
	accountingsync.ItemRoleShortPayWriteOff: "Short-pay write-off",
}

var termLabels = map[string]string{
	string(customer.PaymentTermNet10):        "Net 10",
	string(customer.PaymentTermNet15):        "Net 15",
	string(customer.PaymentTermNet30):        "Net 30",
	string(customer.PaymentTermNet45):        "Net 45",
	string(customer.PaymentTermNet60):        "Net 60",
	string(customer.PaymentTermNet90):        "Net 90",
	string(customer.PaymentTermDueOnReceipt): "Due on receipt",
}

var paymentMethodLabels = map[string]string{
	string(customerpayment.MethodACH):   "ACH",
	string(customerpayment.MethodCheck): "Check",
	string(customerpayment.MethodWire):  "Wire",
	string(customerpayment.MethodCard):  "Card",
	string(customerpayment.MethodCash):  "Cash",
	string(customerpayment.MethodOther): "Other",
}

func keyLabel(targetType accountingsync.MappingTargetType, key string) string {
	switch targetType {
	case accountingsync.TargetLineType:
		return lineTypeLabels[key]
	case accountingsync.TargetItemRole:
		return itemRoleLabels[key]
	case accountingsync.TargetPaymentTerm:
		return termLabels[key]
	case accountingsync.TargetPaymentMethod:
		return paymentMethodLabels[key]
	case accountingsync.TargetAccountRole:
		return roleLabels[key]
	case accountingsync.TargetAccessorialCharge,
		accountingsync.TargetCustomer,
		accountingsync.TargetCarrier,
		accountingsync.TargetDriver,
		accountingsync.TargetGLAccount:
		return key
	default:
		return key
	}
}

var termDueDays = map[string]int{
	string(customer.PaymentTermNet10):        10,
	string(customer.PaymentTermNet15):        15,
	string(customer.PaymentTermNet30):        30,
	string(customer.PaymentTermNet45):        45,
	string(customer.PaymentTermNet60):        60,
	string(customer.PaymentTermNet90):        90,
	string(customer.PaymentTermDueOnReceipt): 0,
}

func MappingIdentity(m *accountingmapping.AccountingMapping) string {
	if m.TargetType.KeyedByObject() {
		return string(m.TargetType) + "|" + m.TrenovaObjectID.String()
	}
	return string(m.TargetType) + "|" + m.TrenovaKey
}

func offeredTargets(
	profile *accountingprovider.ProviderProfile,
	all []*accountingmappingscore.Target,
) []*accountingmappingscore.Target {
	offered := make([]*accountingmappingscore.Target, 0, len(all))
	for _, t := range all {
		if !profile.OffersKey(t.TargetType, t.Key) {
			continue
		}
		t.Kind = profile.KindFor(t.TargetType)
		offered = append(offered, t)
	}
	return offered
}

func roleAccounts(control *accountingcontrol.AccountingControl) map[string]pulid.ID {
	if control == nil {
		return map[string]pulid.ID{}
	}
	return map[string]pulid.ID{
		accountingsync.AccountRoleAR:                      control.DefaultARAccountID,
		accountingsync.AccountRoleRevenue:                 control.DefaultRevenueAccountID,
		accountingsync.AccountRoleDeposit:                 control.DefaultCashAccountID,
		accountingsync.AccountRoleWriteOff:                control.DefaultWriteOffAccountID,
		accountingsync.AccountRoleAP:                      control.DefaultAPAccountID,
		accountingsync.AccountRolePurchasedTransportation: control.DefaultPurchasedTransportationAccountID,
	}
}

func keyTargets() []*accountingmappingscore.Target {
	targets := make([]*accountingmappingscore.Target, 0, 16)
	for _, key := range accountingsync.TargetLineType.Keys() {
		targets = append(targets, &accountingmappingscore.Target{
			TargetType: accountingsync.TargetLineType,
			Key:        key,
			Label:      keyLabel(accountingsync.TargetLineType, key),
			Names:      accountingmappingscore.LineTypeSynonyms[key],
		})
	}
	for _, key := range accountingsync.TargetItemRole.Keys() {
		targets = append(targets, &accountingmappingscore.Target{
			TargetType: accountingsync.TargetItemRole,
			Key:        key,
			Label:      keyLabel(accountingsync.TargetItemRole, key),
			Names:      accountingmappingscore.ItemRoleSynonyms[key],
		})
	}
	for _, key := range accountingsync.TargetPaymentTerm.Keys() {
		days := termDueDays[key]
		targets = append(targets, &accountingmappingscore.Target{
			TargetType: accountingsync.TargetPaymentTerm,
			Key:        key,
			Label:      keyLabel(accountingsync.TargetPaymentTerm, key),
			Names:      accountingmappingscore.TermSynonyms[key],
			DueDays:    &days,
		})
	}
	for _, key := range accountingsync.TargetPaymentMethod.Keys() {
		targets = append(targets, &accountingmappingscore.Target{
			TargetType: accountingsync.TargetPaymentMethod,
			Key:        key,
			Label:      keyLabel(accountingsync.TargetPaymentMethod, key),
			Names:      accountingmappingscore.PaymentMethodSynonyms[key],
		})
	}
	return targets
}

func activeFilter(tenantInfo pagination.TenantInfo, offset int) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: tenantInfo,
		Pagination: pagination.Info{Limit: targetPageSize, Offset: offset},
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "status", Operator: dbtype.OpEqual, Value: "Active"},
		},
		Sort: []domaintypes.SortField{{Field: "id", Direction: dbtype.SortDirectionAsc}},
	}
}

func accessorialTarget(charge *accessorialcharge.AccessorialCharge) *accountingmappingscore.Target {
	return &accountingmappingscore.Target{
		TargetType: accountingsync.TargetAccessorialCharge,
		ObjectID:   charge.ID,
		Label:      joinLabel(charge.Code, charge.Description),
		Code:       charge.Code,
		Names:      nonEmpty(charge.Description, charge.Code),
	}
}

func customerTarget(cus *customer.Customer) *accountingmappingscore.Target {
	return &accountingmappingscore.Target{
		TargetType:  accountingsync.TargetCustomer,
		ObjectID:    cus.ID,
		Label:       joinLabel(cus.Name, cus.Code),
		Names:       nonEmpty(cus.Name),
		PostalCode:  cus.PostalCode,
		Identifiers: nonEmpty(cus.MCNumber, cus.DOTNumber),
	}
}

func carrierTarget(carr *carrier.Carrier) *accountingmappingscore.Target {
	return &accountingmappingscore.Target{
		TargetType:  accountingsync.TargetCarrier,
		ObjectID:    carr.ID,
		Label:       joinLabel(carr.Name, carr.SCAC),
		Names:       nonEmpty(carr.Name, carr.DBAName),
		PostalCode:  carr.PostalCode,
		Identifiers: nonEmpty(carr.MCNumber, carr.DOTNumber),
	}
}

func driverTarget(wrk *worker.Worker) *accountingmappingscore.Target {
	name := WorkerName(wrk)
	return &accountingmappingscore.Target{
		TargetType: accountingsync.TargetDriver,
		ObjectID:   wrk.ID,
		Label:      name,
		Names:      nonEmpty(name),
		PostalCode: wrk.PostalCode,
	}
}

func WorkerName(wrk *worker.Worker) string {
	return strings.TrimSpace(
		strings.TrimSpace(wrk.FirstName) + " " + strings.TrimSpace(wrk.LastName),
	)
}

func glAccountTarget(account *glaccount.GLAccount) *accountingmappingscore.Target {
	return &accountingmappingscore.Target{
		TargetType: accountingsync.TargetGLAccount,
		ObjectID:   account.ID,
		Label:      joinLabel(account.AccountCode, account.Name),
		Code:       account.AccountCode,
		Names:      nonEmpty(account.Name),
	}
}

func joinLabel(primary, secondary string) string {
	primary, secondary = strings.TrimSpace(primary), strings.TrimSpace(secondary)
	switch {
	case secondary == "" || strings.EqualFold(primary, secondary):
		return primary
	case primary == "":
		return secondary
	default:
		return primary + " (" + secondary + ")"
	}
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
