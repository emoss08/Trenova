package accountingmappingservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/worker"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
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

func (t *target) identity() string {
	if t.TargetType.KeyedByObject() {
		return string(t.TargetType) + "|" + t.ObjectID.String()
	}
	return string(t.TargetType) + "|" + t.Key
}

func mappingIdentity(m *accountingsync.AccountingMapping) string {
	if m.TargetType.KeyedByObject() {
		return string(m.TargetType) + "|" + m.TrenovaObjectID.String()
	}
	return string(m.TargetType) + "|" + m.TrenovaKey
}

func (s *Service) listTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*target, error) {
	targets := make([]*target, 0, 64)

	roles, err := s.roleTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, roles...)
	targets = append(targets, keyTargets()...)

	charges, err := s.accessorialTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, charges...)

	customers, err := s.customerTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, customers...)

	carriers, err := s.carrierTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, carriers...)

	drivers, err := s.driverTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, drivers...)

	accounts, err := s.glAccountTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, accounts...)

	return targets, nil
}

func (s *Service) roleTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*target, error) {
	control, err := s.accountingControls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	accountFor := roleAccounts(control)
	ids := make([]pulid.ID, 0, len(accountFor))
	for _, id := range accountFor {
		if !id.IsNil() {
			ids = append(ids, id)
		}
	}

	names := make(map[pulid.ID][2]string, len(ids))
	if len(ids) > 0 {
		accounts, getErr := s.glAccounts.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
			TenantInfo:   tenantInfo,
			GLAccountIDs: ids,
		})
		if getErr != nil {
			return nil, getErr
		}
		for _, account := range accounts {
			names[account.ID] = [2]string{account.Name, account.AccountCode}
		}
	}

	targets := make([]*target, 0, len(accountingsync.TargetAccountRole.Keys()))
	for _, role := range accountingsync.TargetAccountRole.Keys() {
		t := &target{
			TargetType: accountingsync.TargetAccountRole,
			Key:        role,
			Label:      roleLabels[role],
			Names:      roleSynonyms[role],
		}
		if account, ok := names[accountFor[role]]; ok {
			t.Names = append([]string{account[0]}, roleSynonyms[role]...)
			t.Code = account[1]
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func roleAccounts(control *tenant.AccountingControl) map[string]pulid.ID {
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

func keyTargets() []*target {
	targets := make([]*target, 0, 16)
	for _, key := range accountingsync.TargetLineType.Keys() {
		targets = append(targets, &target{
			TargetType: accountingsync.TargetLineType,
			Key:        key,
			Label:      keyLabel(accountingsync.TargetLineType, key),
			Names:      lineTypeSynonyms[key],
		})
	}
	for _, key := range accountingsync.TargetItemRole.Keys() {
		targets = append(targets, &target{
			TargetType: accountingsync.TargetItemRole,
			Key:        key,
			Label:      keyLabel(accountingsync.TargetItemRole, key),
			Names:      itemRoleSynonyms[key],
		})
	}
	for _, key := range accountingsync.TargetPaymentTerm.Keys() {
		days := termDueDays[key]
		targets = append(targets, &target{
			TargetType: accountingsync.TargetPaymentTerm,
			Key:        key,
			Label:      keyLabel(accountingsync.TargetPaymentTerm, key),
			Names:      termSynonyms[key],
			DueDays:    &days,
		})
	}
	for _, key := range accountingsync.TargetPaymentMethod.Keys() {
		targets = append(targets, &target{
			TargetType: accountingsync.TargetPaymentMethod,
			Key:        key,
			Label:      keyLabel(accountingsync.TargetPaymentMethod, key),
			Names:      paymentMethodSynonyms[key],
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

func (s *Service) accessorialTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*target, error) {
	targets := make([]*target, 0)
	for offset := 0; ; offset += targetPageSize {
		page, err := s.accessorials.List(ctx, &repositories.ListAccessorialChargeRequest{
			Filter: activeFilter(tenantInfo, offset),
		})
		if err != nil {
			return nil, err
		}
		for _, charge := range page.Items {
			targets = append(targets, accessorialTarget(charge))
		}
		if len(page.Items) < targetPageSize || offset+len(page.Items) >= page.Total {
			return targets, nil
		}
	}
}

func accessorialTarget(charge *accessorialcharge.AccessorialCharge) *target {
	return &target{
		TargetType: accountingsync.TargetAccessorialCharge,
		ObjectID:   charge.ID,
		Label:      joinLabel(charge.Code, charge.Description),
		Code:       charge.Code,
		Names:      nonEmpty(charge.Description, charge.Code),
	}
}

func (s *Service) customerTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*target, error) {
	targets := make([]*target, 0)
	for offset := 0; ; offset += targetPageSize {
		page, err := s.customers.List(ctx, &repositories.ListCustomerRequest{
			Filter: activeFilter(tenantInfo, offset),
		})
		if err != nil {
			return nil, err
		}
		for _, cus := range page.Items {
			targets = append(targets, customerTarget(cus))
		}
		if len(page.Items) < targetPageSize || offset+len(page.Items) >= page.Total {
			return targets, nil
		}
	}
}

func customerTarget(cus *customer.Customer) *target {
	return &target{
		TargetType:  accountingsync.TargetCustomer,
		ObjectID:    cus.ID,
		Label:       joinLabel(cus.Name, cus.Code),
		Names:       nonEmpty(cus.Name),
		PostalCode:  cus.PostalCode,
		Identifiers: nonEmpty(cus.MCNumber, cus.DOTNumber),
	}
}

func (s *Service) carrierTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*target, error) {
	targets := make([]*target, 0)
	for offset := 0; ; offset += targetPageSize {
		page, err := s.carriers.List(ctx, &repositories.ListCarrierRequest{
			Filter: activeFilter(tenantInfo, offset),
		})
		if err != nil {
			return nil, err
		}
		for _, carr := range page.Items {
			targets = append(targets, carrierTarget(carr))
		}
		if len(page.Items) < targetPageSize || offset+len(page.Items) >= page.Total {
			return targets, nil
		}
	}
}

func carrierTarget(carr *carrier.Carrier) *target {
	return &target{
		TargetType:  accountingsync.TargetCarrier,
		ObjectID:    carr.ID,
		Label:       joinLabel(carr.Name, carr.SCAC),
		Names:       nonEmpty(carr.Name, carr.DBAName),
		PostalCode:  carr.PostalCode,
		Identifiers: nonEmpty(carr.MCNumber, carr.DOTNumber),
	}
}

func (s *Service) driverTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*target, error) {
	workers, err := s.mappings.ListOwnerOperators(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets := make([]*target, 0, len(workers))
	for _, wrk := range workers {
		targets = append(targets, driverTarget(wrk))
	}
	return targets, nil
}

func driverTarget(wrk *worker.Worker) *target {
	name := workerName(wrk)
	return &target{
		TargetType: accountingsync.TargetDriver,
		ObjectID:   wrk.ID,
		Label:      name,
		Names:      nonEmpty(name),
		PostalCode: wrk.PostalCode,
	}
}

func workerName(wrk *worker.Worker) string {
	return strings.TrimSpace(
		strings.TrimSpace(wrk.FirstName) + " " + strings.TrimSpace(wrk.LastName),
	)
}

func (s *Service) glAccountTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*target, error) {
	ids, err := s.payablesAccountIDs(ctx, tenantInfo)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	accounts, err := s.glAccounts.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
		TenantInfo:   tenantInfo,
		GLAccountIDs: ids,
	})
	if err != nil {
		return nil, err
	}
	targets := make([]*target, 0, len(accounts))
	for _, account := range accounts {
		targets = append(targets, glAccountTarget(account))
	}
	return targets, nil
}

func glAccountTarget(account *glaccount.GLAccount) *target {
	return &target{
		TargetType: accountingsync.TargetGLAccount,
		ObjectID:   account.ID,
		Label:      joinLabel(account.AccountCode, account.Name),
		Code:       account.AccountCode,
		Names:      nonEmpty(account.Name),
	}
}

func (s *Service) payablesAccountIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]pulid.ID, error) {
	control, err := s.accountingControls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	covered := make(map[pulid.ID]struct{}, 8)
	for _, id := range roleAccounts(control) {
		covered[id] = struct{}{}
	}

	candidates := make([]pulid.ID, 0, 16)
	if control != nil {
		candidates = append(candidates,
			control.DefaultDriverPayExpenseAccountID,
			control.DefaultDriverReimbursementAccountID,
			control.DefaultSettlementsPayableAccountID,
			control.DefaultDriverAdvanceAccountID,
			control.DefaultEscrowLiabilityAccountID,
		)
	}
	if s.settlementControls != nil {
		carrierControl, getErr := s.settlementControls.GetOrCreate(ctx, tenantInfo)
		if getErr != nil {
			return nil, getErr
		}
		candidates = append(candidates,
			pulid.ConvertFromPtr(carrierControl.DefaultAPAccountID),
			pulid.ConvertFromPtr(carrierControl.DefaultPurchasedTransportationAccountID),
		)
	}
	if s.payCodes != nil {
		codes, listErr := s.payCodes.ListActive(ctx, repositories.ListActivePayCodesRequest{
			TenantInfo: tenantInfo,
		})
		if listErr != nil {
			return nil, listErr
		}
		for _, code := range codes {
			candidates = append(candidates, pulid.ConvertFromPtr(code.GLAccountID))
		}
	}

	ids := make([]pulid.ID, 0, len(candidates))
	for _, id := range candidates {
		if id.IsNil() {
			continue
		}
		if _, seen := covered[id]; seen {
			continue
		}
		covered[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
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
