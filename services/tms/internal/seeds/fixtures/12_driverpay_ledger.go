package fixtures

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol/costingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	ledgerAccountDriverPayExpense  = "5010"
	ledgerAccountPurchasedTrans    = "5020"
	ledgerAccountDriverBenefits    = "5040"
	ledgerAccountSettlementPayable = "2120"
	ledgerAccountDriverAdvances    = "1140"
	ledgerAccountEscrowLiability   = "2150"
)

type DriverPayLedgerSeed struct {
}

var DriverPayLedger = Seeds.Add(seed.Seed{
	Name:        "driver_pay_ledger",
	Kind:        seed.Fixture,
	Description: "Wires driver settlement GL accounts, cost-control sourcing, recurring earnings, and journals for seeded settlements",
	After:       []*seed.Seed{AdminAccount, SystemAccount, DriverPay},
	Run:         (&DriverPayLedgerSeed{}).Run,
})

func (s *DriverPayLedgerSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	accounts, err := s.ensureLedgerAccounts(ctx, db, sr, org.ID, org.BusinessUnitID)
	if err != nil {
		return fmt.Errorf("ensure ledger accounts: %w", err)
	}

	if err = s.configureAccountingControl(
		ctx,
		db,
		org.ID,
		org.BusinessUnitID,
		accounts,
	); err != nil {
		return fmt.Errorf("configure accounting control: %w", err)
	}

	if err = s.configureCostCategories(ctx, db, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("configure cost categories: %w", err)
	}

	if err = s.ensureRecurringEarnings(
		ctx,
		db,
		sr,
		org.ID,
		org.BusinessUnitID,
		admin.ID,
	); err != nil {
		return fmt.Errorf("ensure recurring earnings: %w", err)
	}

	if err = s.postSeededSettlementJournals(
		ctx,
		db,
		sr,
		org.ID,
		org.BusinessUnitID,
		admin.ID,
		accounts,
	); err != nil {
		return fmt.Errorf("post seeded settlement journals: %w", err)
	}

	return nil
}

// -- GL accounts --

func (s *DriverPayLedgerSeed) ensureLedgerAccounts(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	accounts, created, err := ensureGLAccounts(ctx, db, &seedGLAccountsRequest{
		orgID: orgID,
		buID:  buID,
		accounts: []seedGLAccount{
			{
				code:        ledgerAccountDriverAdvances,
				name:        "Driver Advances Receivable",
				description: "Outstanding cash and money-code advances owed back by drivers, recovered through settlements",
				category:    "Asset",
			},
			{
				code:        ledgerAccountEscrowLiability,
				name:        "Driver Escrow Liability",
				description: "Driver maintenance escrow balances held on behalf of owner-operators per 49 CFR 376.12(k)",
				category:    "Liability",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	required := []string{
		ledgerAccountDriverPayExpense,
		ledgerAccountPurchasedTrans,
		ledgerAccountDriverBenefits,
		ledgerAccountSettlementPayable,
	}
	for _, code := range required {
		if _, ok := accounts[code]; !ok {
			return nil, fmt.Errorf("required gl account %s not found", code)
		}
	}

	if created > 0 {
	}

	return accounts, nil
}

func (s *DriverPayLedgerSeed) configureAccountingControl(
	ctx context.Context,
	db *keel.DB,
	orgID, buID pulid.ID,
	accounts map[string]pulid.ID,
) error {
	control, err := tenantdb.AccountingControls.AllTenants().Where(
		tenantdb.AccountingControls.OrganizationID.Eq(orgID),
		tenantdb.AccountingControls.BusinessUnitID.Eq(buID),
	).First(ctx, db)
	if err != nil {
		if errors.Is(err, keel.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get accounting control: %w", err)
	}
	if !control.DefaultDriverPayExpenseAccountID.IsNil() {
		return nil
	}

	ac := tenantdb.AccountingControls
	if _, err = ac.AllTenants().Where(
		ac.OrganizationID.Eq(orgID),
		ac.BusinessUnitID.Eq(buID),
	).Update(
		ctx,
		db,
		ac.DefaultDriverPayExpenseAccountID.To(accounts[ledgerAccountDriverPayExpense]),
		ac.DefaultPurchasedTransportationAccountID.To(accounts[ledgerAccountPurchasedTrans]),
		ac.DefaultSettlementsPayableAccountID.To(accounts[ledgerAccountSettlementPayable]),
		ac.DefaultDriverAdvanceAccountID.To(accounts[ledgerAccountDriverAdvances]),
		ac.DefaultEscrowLiabilityAccountID.To(accounts[ledgerAccountEscrowLiability]),
		ac.DefaultDriverReimbursementAccountID.To(accounts[ledgerAccountDriverBenefits]),
	); err != nil {
		return fmt.Errorf("update accounting control: %w", err)
	}

	return nil
}

func (s *DriverPayLedgerSeed) configureCostCategories(
	ctx context.Context,
	db *keel.DB,
	orgID, buID pulid.ID,
) error {
	cc := costingcontroldb.CostCategories
	if _, err := cc.AllTenants().Where(
		cc.OrganizationID.Eq(orgID),
		cc.BusinessUnitID.Eq(buID),
		cc.Category.In(
			costingcontrol.CategoryTypeDriverWages,
			costingcontrol.CategoryTypeDriverBenefits,
		),
		cc.RateSource.Eq(costingcontrol.RateSourceBenchmark),
	).Update(ctx, db, cc.RateSource.To(costingcontrol.RateSourceGLActual)); err != nil {
		return fmt.Errorf("update cost categories: %w", err)
	}
	return nil
}

// -- Recurring earnings --

func (s *DriverPayLedgerSeed) ensureRecurringEarnings(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
	adminID pulid.ID,
) error {
	payCodes, err := ensurePayCodes(ctx, db, orgID, buID)
	if err != nil {
		return fmt.Errorf("ensure pay codes: %w", err)
	}

	workers, err := workerdb.Workers.AllTenants().Pick(workerdb.Workers.ID, workerdb.Workers.Email).Where(
		workerdb.Workers.OrganizationID.Eq(orgID),
		workerdb.Workers.BusinessUnitID.Eq(buID),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("get workers: %w", err)
	}
	workersByEmail := make(map[string]pulid.ID, len(workers))
	for i := range workers {
		workersByEmail[workers[i].Email] = workers[i].ID
	}

	now := timeutils.NowUnix()
	day := int64(86400)

	defs := []struct {
		workerKey   string
		code        string
		status      driverpay.EarningStatus
		frequency   driverpay.EarningFrequency
		description string
		amountMinor int64
		capMinor    int64
		paidMinor   int64
	}{
		{
			payWorkerJohn,
			"PERDIEM",
			driverpay.EarningStatusActive,
			driverpay.EarningFrequencyEverySettlement,
			"OTR per diem — IRS substantiated M&IE",
			33250,
			0,
			299250,
		},
		{
			payWorkerRobert,
			"PERDIEM",
			driverpay.EarningStatusActive,
			driverpay.EarningFrequencyEverySettlement,
			"OTR per diem — IRS substantiated M&IE",
			33250,
			0,
			199500,
		},
		{
			payWorkerEmily,
			"PERDIEM",
			driverpay.EarningStatusActive,
			driverpay.EarningFrequencyEverySettlement,
			"Regional per diem — partial-day M&IE",
			19950,
			0,
			119700,
		},
		{
			payWorkerJane,
			"SAFETY",
			driverpay.EarningStatusActive,
			driverpay.EarningFrequencyMonthly,
			"Quarterly safety bonus accrual — clean CSA record",
			12500,
			0,
			62500,
		},
		{
			payWorkerSarah,
			"STIPEND",
			driverpay.EarningStatusActive,
			driverpay.EarningFrequencyMonthly,
			"Cell phone and ELD data stipend",
			5000,
			0,
			25000,
		},
		{
			payWorkerMike,
			"PERFORM",
			driverpay.EarningStatusActive,
			driverpay.EarningFrequencyEverySettlement,
			"On-time delivery bonus program",
			7500,
			195000,
			97500,
		},
		{
			payWorkerCarlos,
			"EQUIPRENT",
			driverpay.EarningStatusActive,
			driverpay.EarningFrequencyEverySettlement,
			"APU rental — company use of driver-owned unit",
			6000,
			0,
			78000,
		},
		{
			payWorkerDavid,
			"LONGEVITY",
			driverpay.EarningStatusPaused,
			driverpay.EarningFrequencyMonthly,
			"Longevity bonus — 3+ years of service",
			10000,
			0,
			30000,
		},
	}

	rows := make([]*driverpay.RecurringEarning, 0, len(defs))
	for _, def := range defs {
		workerID, ok := workersByEmail[def.workerKey]
		if !ok {
			continue
		}
		payCodeID, hasCode := payCodes["Earning/"+def.code]
		if !hasCode {
			return fmt.Errorf("pay code %s not found", def.code)
		}
		earning := &driverpay.RecurringEarning{
			ID:              pulid.MustNew("rern_"),
			BusinessUnitID:  buID,
			OrganizationID:  orgID,
			WorkerID:        workerID,
			PayCodeID:       payCodeID,
			Status:          def.status,
			Frequency:       def.frequency,
			Description:     def.description,
			AmountMinor:     def.amountMinor,
			PaidToDateMinor: def.paidMinor,
			StartDate:       now - (270 * day),
			CurrencyCode:    "USD",
			CreatedByID:     adminID,
			Kind:            driverpay.EarningKindStandard,
		}
		if def.capMinor > 0 {
			capMinor := def.capMinor
			earning.TotalCapMinor = &capMinor
		}
		rows = append(rows, earning)
	}

	if err = driverpaydb.RecurringEarnings.CreateMany(ctx, db, rows); err != nil {
		return fmt.Errorf("insert recurring earnings: %w", err)
	}

	return nil
}

// -- Settlement journals --

func (s *DriverPayLedgerSeed) postSeededSettlementJournals(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
	adminID pulid.ID,
	accounts map[string]pulid.ID,
) error {
	settlements, err := driversettlementdb.Settlements.AllTenants().Load(driversettlementdb.Settlements.Lines()).Where(
		driversettlementdb.Settlements.OrganizationID.Eq(orgID),
		driversettlementdb.Settlements.BusinessUnitID.Eq(buID),
		driversettlementdb.Settlements.Status.In(driversettlement.StatusPosted,
			driversettlement.StatusPaid),
		driversettlementdb.Settlements.PostedJournalBatchID.IsNull(),
	).Order(driversettlementdb.Settlements.SettlementNumber.Asc()).All(ctx, db)
	if err != nil {
		return fmt.Errorf("get unposted settlements: %w", err)
	}

	posted := 0
	for idx, settlement := range settlements {
		accountingDate := timeutils.NowUnix()
		if settlement.PostedAt != nil {
			accountingDate = *settlement.PostedAt
		}

		expenseAccountID := accounts[ledgerAccountDriverPayExpense]
		if settlement.Classification == driverpay.PayeeClassificationOwnerOperator {
			expenseAccountID = accounts[ledgerAccountPurchasedTrans]
		}
		legs := driversettlementservice.BuildSettlementPostingLegs(
			settlement,
			&driversettlementservice.PostingAccounts{
				Expense:       expenseAccountID,
				Reimbursement: accounts[ledgerAccountDriverBenefits],
				Payable:       accounts[ledgerAccountSettlementPayable],
				Advance:       accounts[ledgerAccountDriverAdvances],
				Escrow:        accounts[ledgerAccountEscrowLiability],
			},
			make(map[pulid.ID]pulid.ID),
		)
		if len(legs) == 0 {
			continue
		}

		journalLegs := make([]seedJournalLeg, 0, len(legs))
		for _, leg := range legs {
			journalLegs = append(journalLegs, seedJournalLeg{
				AccountID: leg.AccountID,
				Debit:     leg.Debit,
				Credit:    leg.Credit,
			})
		}
		suffix := fmt.Sprintf("%04d", idx+1)
		journal, err := postSeedJournal(ctx, db, &seedJournalRequest{
			orgID:          orgID,
			buID:           buID,
			actorID:        adminID,
			accountingDate: accountingDate,
			batchNumber:    "SEED-JB-" + suffix,
			entryNumber:    "SEED-JE-" + suffix,
			description:    "Driver settlement " + settlement.SettlementNumber,
			sourceType:     "DriverSettlement",
			sourceID:       settlement.ID.String(),
			documentNumber: settlement.SettlementNumber,
			event:          tenant.JournalSourceEventDriverSettlementPosted.String(),
			idempotencyKey: "driver-settlement-posted:" + settlement.ID.String(),
			legs:           journalLegs,
		})
		if errors.Is(err, errSeedNoFiscalPeriod) {
			continue
		}
		if err != nil {
			return err
		}

		st := driversettlementdb.Settlements
		if _, err = st.AllTenants().Where(
			st.ID.Eq(settlement.ID),
			st.OrganizationID.Eq(orgID),
			st.BusinessUnitID.Eq(buID),
		).Update(ctx, db, st.PostedJournalBatchID.To(journal.batchID)); err != nil {
			return fmt.Errorf("link settlement to journal batch: %w", err)
		}
		posted++
	}

	sr.Count(int64(posted))
	return nil
}
