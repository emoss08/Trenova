package accountingmappingtargets

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingscore"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (c *Catalog) WorkerWithState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.Worker, error) {
	if c.workers == nil {
		return nil, errortypes.NewBusinessError("Driver records are not available")
	}
	wrk, err := c.workers.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:           workerID,
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError(
				"The driver for this mapping no longer exists",
			)
		}
		return nil, err
	}
	return wrk, nil
}

func (c *Catalog) TargetFor(
	ctx context.Context,
	req *accountingmappingports.EnsureAccountingMappingRequest,
) (*accountingmappingscore.Target, error) {
	switch req.TargetType {
	case accountingsync.TargetCustomer:
		cus, err := c.customers.GetByID(ctx, customerports.GetCustomerByIDRequest{
			ID:         req.ObjectID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		return customerTarget(cus), nil
	case accountingsync.TargetAccessorialCharge:
		tenantInfo := req.TenantInfo
		charge, err := c.accessorials.GetByID(
			ctx,
			accessorialchargeports.GetAccessorialChargeByIDRequest{
				ID:         req.ObjectID,
				TenantInfo: &tenantInfo,
			},
		)
		if err != nil {
			return nil, err
		}
		return accessorialTarget(charge), nil
	case accountingsync.TargetAccountRole,
		accountingsync.TargetLineType,
		accountingsync.TargetItemRole,
		accountingsync.TargetPaymentTerm,
		accountingsync.TargetPaymentMethod:
		if !req.TargetType.AcceptsKey(req.Key) {
			return nil, errortypes.NewValidationError(
				"key",
				errortypes.ErrInvalid,
				"{0} is not a {1} key",
				req.Key,
				string(req.TargetType),
			)
		}
		return &accountingmappingscore.Target{
			TargetType: req.TargetType,
			Key:        req.Key,
			Label:      keyLabel(req.TargetType, req.Key),
		}, nil
	case accountingsync.TargetCarrier:
		carr, err := c.carriers.GetByID(ctx, carrierports.GetCarrierByIDRequest{
			ID:         req.ObjectID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		return carrierTarget(carr), nil
	case accountingsync.TargetDriver:
		wrk, err := c.WorkerWithState(ctx, req.TenantInfo, req.ObjectID)
		if err != nil {
			return nil, err
		}
		return driverTarget(wrk), nil
	case accountingsync.TargetGLAccount:
		accounts, err := c.glAccounts.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
			TenantInfo:   req.TenantInfo,
			GLAccountIDs: []pulid.ID{req.ObjectID},
		})
		if err != nil {
			return nil, err
		}
		if len(accounts) == 0 {
			return nil, errortypes.NewNotFoundError("The GL account no longer exists")
		}
		return glAccountTarget(accounts[0]), nil
	default:
		return nil, errortypes.NewValidationError(
			"targetType",
			errortypes.ErrInvalid,
			"{0} is not a mapping target",
			string(req.TargetType),
		)
	}
}

func (c *Catalog) ListTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	conn *accountingconnection.AccountingConnection,
) ([]*accountingmappingscore.Target, error) {
	all, err := c.allTargets(ctx, tenantInfo, conn.SendsLedger())
	if err != nil {
		return nil, err
	}
	profile := accountingprovider.MustProfile(conn.IntegrationType)
	return offeredTargets(&profile, all), nil
}

func (c *Catalog) allTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ledger bool,
) ([]*accountingmappingscore.Target, error) {
	targets := make([]*accountingmappingscore.Target, 0, 64)

	roles, err := c.roleTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, roles...)
	targets = append(targets, keyTargets()...)

	charges, err := c.accessorialTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, charges...)

	customers, err := c.customerTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, customers...)

	carriers, err := c.carrierTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, carriers...)

	drivers, err := c.driverTargets(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets = append(targets, drivers...)

	accounts, err := c.glAccountTargets(ctx, tenantInfo, ledger)
	if err != nil {
		return nil, err
	}
	targets = append(targets, accounts...)

	return targets, nil
}

func (c *Catalog) roleTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*accountingmappingscore.Target, error) {
	control, err := c.accountingControls.GetByOrgID(ctx, tenantInfo.OrgID)
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
		accounts, getErr := c.glAccounts.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
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

	targets := make(
		[]*accountingmappingscore.Target,
		0,
		len(accountingsync.TargetAccountRole.Keys()),
	)
	for _, role := range accountingsync.TargetAccountRole.Keys() {
		t := &accountingmappingscore.Target{
			TargetType: accountingsync.TargetAccountRole,
			Key:        role,
			Label:      roleLabels[role],
			Names:      accountingmappingscore.RoleSynonyms[role],
		}
		if account, ok := names[accountFor[role]]; ok {
			t.Names = append([]string{account[0]}, accountingmappingscore.RoleSynonyms[role]...)
			t.Code = account[1]
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func (c *Catalog) accessorialTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*accountingmappingscore.Target, error) {
	targets := make([]*accountingmappingscore.Target, 0)
	for offset := 0; ; offset += targetPageSize {
		page, err := c.accessorials.List(ctx, &accessorialchargeports.ListAccessorialChargeRequest{
			Filter: activeFilter(tenantInfo, offset),
		})
		if err != nil {
			return nil, err
		}
		for _, charge := range page.Items {
			targets = append(targets, accessorialTarget(charge))
		}
		if len(page.Items) < targetPageSize ||
			(page.TotalExact && offset+len(page.Items) >= page.Total) {
			return targets, nil
		}
	}
}

func (c *Catalog) customerTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*accountingmappingscore.Target, error) {
	targets := make([]*accountingmappingscore.Target, 0)
	for offset := 0; ; offset += targetPageSize {
		page, err := c.customers.List(ctx, &customerports.ListCustomerRequest{
			Filter: activeFilter(tenantInfo, offset),
		})
		if err != nil {
			return nil, err
		}
		for _, cus := range page.Items {
			targets = append(targets, customerTarget(cus))
		}
		if len(page.Items) < targetPageSize ||
			(page.TotalExact && offset+len(page.Items) >= page.Total) {
			return targets, nil
		}
	}
}

func (c *Catalog) carrierTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*accountingmappingscore.Target, error) {
	targets := make([]*accountingmappingscore.Target, 0)
	for offset := 0; ; offset += targetPageSize {
		page, err := c.carriers.List(ctx, &carrierports.ListCarrierRequest{
			Filter: activeFilter(tenantInfo, offset),
		})
		if err != nil {
			return nil, err
		}
		for _, carr := range page.Items {
			targets = append(targets, carrierTarget(carr))
		}
		if len(page.Items) < targetPageSize ||
			(page.TotalExact && offset+len(page.Items) >= page.Total) {
			return targets, nil
		}
	}
}

func (c *Catalog) driverTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*accountingmappingscore.Target, error) {
	workers, err := c.mappings.ListOwnerOperators(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	targets := make([]*accountingmappingscore.Target, 0, len(workers))
	for _, wrk := range workers {
		targets = append(targets, driverTarget(wrk))
	}
	return targets, nil
}

func (c *Catalog) glAccountTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ledger bool,
) ([]*accountingmappingscore.Target, error) {
	ids, err := c.payablesAccountIDs(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	accounts := make([]*glaccount.GLAccount, 0, len(ids))
	if len(ids) > 0 {
		if accounts, err = c.glAccounts.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
			TenantInfo:   tenantInfo,
			GLAccountIDs: ids,
		}); err != nil {
			return nil, err
		}
	}
	if ledger {
		active, activeErr := c.activeGLAccounts(ctx, tenantInfo)
		if activeErr != nil {
			return nil, activeErr
		}
		accounts = append(accounts, active...)
	}

	seen := make(map[pulid.ID]struct{}, len(accounts))
	targets := make([]*accountingmappingscore.Target, 0, len(accounts))
	for _, account := range accounts {
		if _, dup := seen[account.ID]; dup {
			continue
		}
		seen[account.ID] = struct{}{}
		targets = append(targets, glAccountTarget(account))
	}
	if !ledger || c.ledger == nil {
		return targets, nil
	}
	posted, err := c.ledger.ListActiveAccounts(ctx, &journalentryports.ListLedgerAccountsRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	for idx := range posted {
		if _, dup := seen[posted[idx].ID]; dup {
			continue
		}
		seen[posted[idx].ID] = struct{}{}
		targets = append(targets, glAccountTarget(&glaccount.GLAccount{
			ID:          posted[idx].ID,
			AccountCode: posted[idx].Code,
			Name:        posted[idx].Name,
		}))
	}
	return targets, nil
}

func (c *Catalog) activeGLAccounts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*glaccount.GLAccount, error) {
	accounts := make([]*glaccount.GLAccount, 0, targetPageSize)
	for offset := 0; ; offset += targetPageSize {
		page, err := c.glAccounts.List(ctx, &glaccountports.ListGLAccountsRequest{
			Filter: activeFilter(tenantInfo, offset),
		})
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, page.Items...)
		if len(page.Items) < targetPageSize ||
			(page.TotalExact && offset+len(page.Items) >= page.Total) {
			return accounts, nil
		}
	}
}

func (c *Catalog) payablesAccountIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]pulid.ID, error) {
	control, err := c.accountingControls.GetByOrgID(ctx, tenantInfo.OrgID)
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
	if c.settlementControls != nil {
		carrierControl, getErr := c.settlementControls.GetOrCreate(ctx, tenantInfo)
		if getErr != nil {
			return nil, getErr
		}
		candidates = append(candidates,
			pulid.ConvertFromPtr(carrierControl.DefaultAPAccountID),
			pulid.ConvertFromPtr(carrierControl.DefaultPurchasedTransportationAccountID),
		)
	}
	if c.payCodes != nil {
		codes, listErr := c.payCodes.ListActive(ctx, driverpayports.ListActivePayCodesRequest{
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

type Deps struct {
	Mappings           accountingmappingports.AccountingMappingRepository
	AccountingControls tenantports.AccountingControlRepository
	GLAccounts         glaccountports.GLAccountRepository
	Ledger             journalentryports.AccountingLedgerSource
	Customers          customerports.CustomerRepository
	Carriers           carrierports.CarrierRepository
	Accessorials       accessorialchargeports.AccessorialChargeRepository
	Workers            workerports.WorkerRepository
	SettlementControls tenantports.CarrierSettlementControlRepository
	PayCodes           driverpayports.PayCodeRepository
}

type Catalog struct {
	mappings           accountingmappingports.AccountingMappingRepository
	accountingControls tenantports.AccountingControlRepository
	glAccounts         glaccountports.GLAccountRepository
	ledger             journalentryports.AccountingLedgerSource
	customers          customerports.CustomerRepository
	carriers           carrierports.CarrierRepository
	accessorials       accessorialchargeports.AccessorialChargeRepository
	workers            workerports.WorkerRepository
	settlementControls tenantports.CarrierSettlementControlRepository
	payCodes           driverpayports.PayCodeRepository
}

//nolint:gocritic // dependency injection
func New(d Deps) *Catalog {
	return &Catalog{
		mappings:           d.Mappings,
		accountingControls: d.AccountingControls,
		glAccounts:         d.GLAccounts,
		ledger:             d.Ledger,
		customers:          d.Customers,
		carriers:           d.Carriers,
		accessorials:       d.Accessorials,
		workers:            d.Workers,
		settlementControls: d.SettlementControls,
		payCodes:           d.PayCodes,
	}
}
