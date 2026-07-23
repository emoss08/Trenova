package development

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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
	seedhelpers.BaseSeed
}

func NewDriverPayLedgerSeed() *DriverPayLedgerSeed {
	seed := &DriverPayLedgerSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DriverPayLedger",
		"1.0.0",
		"Wires driver settlement GL accounts, cost-control sourcing, recurring earnings, and journals for seeded settlements",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)
	seed.SetDependencies(
		seedhelpers.SeedNormalAccount,
		seedhelpers.SeedDriverPay,
	)
	return seed
}

func (s *DriverPayLedgerSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}

			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}

			accounts, err := s.ensureLedgerAccounts(ctx, tx, sc, org.ID, org.BusinessUnitID)
			if err != nil {
				return fmt.Errorf("ensure ledger accounts: %w", err)
			}

			if err = s.configureAccountingControl(ctx, tx, org.ID, org.BusinessUnitID, accounts); err != nil {
				return fmt.Errorf("configure accounting control: %w", err)
			}

			if err = s.configureCostCategories(ctx, tx, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("configure cost categories: %w", err)
			}

			if err = s.ensureRecurringEarnings(ctx, tx, sc, org.ID, org.BusinessUnitID, admin.ID); err != nil {
				return fmt.Errorf("ensure recurring earnings: %w", err)
			}

			if err = s.postSeededSettlementJournals(ctx, tx, sc, org.ID, org.BusinessUnitID, admin.ID, accounts); err != nil {
				return fmt.Errorf("post seeded settlement journals: %w", err)
			}

			return nil
		},
	)
}

// -- GL accounts --

func (s *DriverPayLedgerSeed) ensureLedgerAccounts(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	var existing []glaccount.GLAccount
	if err := tx.NewSelect().Model(&existing).Column("id", "account_code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get gl accounts: %w", err)
	}

	accounts := make(map[string]pulid.ID, len(existing))
	for i := range existing {
		accounts[existing[i].AccountCode] = existing[i].ID
	}

	var accountTypes []struct {
		ID       pulid.ID `bun:"id"`
		Category string   `bun:"category"`
	}
	if err := tx.NewSelect().Table("account_types").Column("id", "category").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx, &accountTypes); err != nil {
		return nil, fmt.Errorf("get account types: %w", err)
	}
	typeByCategory := make(map[string]pulid.ID, len(accountTypes))
	for _, at := range accountTypes {
		typeByCategory[at.Category] = at.ID
	}

	defs := []struct {
		code        string
		name        string
		description string
		category    string
	}{
		{
			ledgerAccountDriverAdvances,
			"Driver Advances Receivable",
			"Outstanding cash and money-code advances owed back by drivers, recovered through settlements",
			"Asset",
		},
		{
			ledgerAccountEscrowLiability,
			"Driver Escrow Liability",
			"Driver maintenance escrow balances held on behalf of owner-operators per 49 CFR 376.12(k)",
			"Liability",
		},
	}

	created := 0
	for _, def := range defs {
		if _, ok := accounts[def.code]; ok {
			continue
		}
		typeID, ok := typeByCategory[def.category]
		if !ok {
			return nil, fmt.Errorf("account type %s not found", def.category)
		}

		account := &glaccount.GLAccount{
			ID:             pulid.MustNew("gla_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			AccountTypeID:  typeID,
			AccountCode:    def.code,
			Name:           def.name,
			Description:    def.description,
			IsSystem:       true,
		}
		if _, err := tx.NewInsert().Model(account).Exec(ctx); err != nil {
			return nil, fmt.Errorf("insert gl account %s: %w", def.code, err)
		}
		if err := sc.TrackCreated(ctx, "gl_accounts", account.ID, s.Name()); err != nil {
			return nil, fmt.Errorf("track gl account: %w", err)
		}
		accounts[def.code] = account.ID
		created++
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
		seedhelpers.LogSuccess(
			"Created driver settlement GL accounts",
			fmt.Sprintf("- Created %d balance-sheet accounts for driver pay", created),
		)
	}

	return accounts, nil
}

func (s *DriverPayLedgerSeed) configureAccountingControl(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
	accounts map[string]pulid.ID,
) error {
	control := new(tenant.AccountingControl)
	err := tx.NewSelect().Model(control).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get accounting control: %w", err)
	}
	if !control.DefaultDriverPayExpenseAccountID.IsNil() {
		return nil
	}

	if _, err = tx.NewUpdate().Model((*tenant.AccountingControl)(nil)).
		Set("default_driver_pay_expense_account_id = ?", accounts[ledgerAccountDriverPayExpense]).
		Set("default_purchased_transportation_account_id = ?", accounts[ledgerAccountPurchasedTrans]).
		Set("default_settlements_payable_account_id = ?", accounts[ledgerAccountSettlementPayable]).
		Set("default_driver_advance_account_id = ?", accounts[ledgerAccountDriverAdvances]).
		Set("default_escrow_liability_account_id = ?", accounts[ledgerAccountEscrowLiability]).
		Set("default_driver_reimbursement_account_id = ?", accounts[ledgerAccountDriverBenefits]).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx); err != nil {
		return fmt.Errorf("update accounting control: %w", err)
	}

	seedhelpers.LogSuccess(
		"Configured driver settlement posting accounts",
		"- Accounting control now maps driver pay, purchased transportation, payable, advance, escrow, and reimbursement accounts",
	)
	return nil
}

func (s *DriverPayLedgerSeed) configureCostCategories(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) error {
	res, err := tx.NewUpdate().Model((*costingcontrol.CostCategory)(nil)).
		Set("rate_source = ?", costingcontrol.RateSourceGLActual).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("category IN (?)", bun.List([]costingcontrol.CategoryType{
			costingcontrol.CategoryTypeDriverWages,
			costingcontrol.CategoryTypeDriverBenefits,
		})).
		Where("rate_source = ?", costingcontrol.RateSourceBenchmark).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update cost categories: %w", err)
	}
	if rows, rowsErr := res.RowsAffected(); rowsErr == nil && rows > 0 {
		seedhelpers.LogSuccess(
			"Switched driver cost categories to GL actuals",
			"- DriverWages and DriverBenefits now compute cost-per-mile from posted settlement journals",
		)
	}
	return nil
}

// -- Recurring earnings --

func (s *DriverPayLedgerSeed) ensureRecurringEarnings(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
	adminID pulid.ID,
) error {
	payCodes, err := ensurePayCodes(ctx, tx, orgID, buID)
	if err != nil {
		return fmt.Errorf("ensure pay codes: %w", err)
	}

	count, err := tx.NewSelect().Model((*driverpay.RecurringEarning)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing recurring earnings: %w", err)
	}
	if count > 0 {
		return nil
	}

	var workers []worker.Worker
	if err = tx.NewSelect().Model(&workers).Column("id", "email").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx); err != nil {
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
		{payWorkerJohn, "PERDIEM", driverpay.EarningStatusActive, driverpay.EarningFrequencyEverySettlement, "OTR per diem — IRS substantiated M&IE", 33250, 0, 299250},
		{payWorkerRobert, "PERDIEM", driverpay.EarningStatusActive, driverpay.EarningFrequencyEverySettlement, "OTR per diem — IRS substantiated M&IE", 33250, 0, 199500},
		{payWorkerEmily, "PERDIEM", driverpay.EarningStatusActive, driverpay.EarningFrequencyEverySettlement, "Regional per diem — partial-day M&IE", 19950, 0, 119700},
		{payWorkerJane, "SAFETY", driverpay.EarningStatusActive, driverpay.EarningFrequencyMonthly, "Quarterly safety bonus accrual — clean CSA record", 12500, 0, 62500},
		{payWorkerSarah, "STIPEND", driverpay.EarningStatusActive, driverpay.EarningFrequencyMonthly, "Cell phone and ELD data stipend", 5000, 0, 25000},
		{payWorkerMike, "PERFORM", driverpay.EarningStatusActive, driverpay.EarningFrequencyEverySettlement, "On-time delivery bonus program", 7500, 195000, 97500},
		{payWorkerCarlos, "EQUIPRENT", driverpay.EarningStatusActive, driverpay.EarningFrequencyEverySettlement, "APU rental — company use of driver-owned unit", 6000, 0, 78000},
		{payWorkerDavid, "LONGEVITY", driverpay.EarningStatusPaused, driverpay.EarningFrequencyMonthly, "Longevity bonus — 3+ years of service", 10000, 0, 30000},
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
		}
		if def.capMinor > 0 {
			capMinor := def.capMinor
			earning.TotalCapMinor = &capMinor
		}
		rows = append(rows, earning)
	}

	if _, err = tx.NewInsert().Model(&rows).Exec(ctx); err != nil {
		return fmt.Errorf("insert recurring earnings: %w", err)
	}
	for _, earning := range rows {
		if err = sc.TrackCreated(ctx, "recurring_earnings", earning.ID, s.Name()); err != nil {
			return fmt.Errorf("track recurring earning: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created recurring earning fixtures",
		fmt.Sprintf("- Created %d recurring earnings (per diem, bonuses, stipends, equipment rental)", len(rows)),
	)
	return nil
}

// -- Settlement journals --

type seedJournalBatch struct {
	bun.BaseModel `bun:"table:journal_batches"`

	ID             pulid.ID `bun:"id,pk"`
	OrganizationID pulid.ID `bun:"organization_id,pk"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk"`
	BatchNumber    string   `bun:"batch_number"`
	BatchType      string   `bun:"batch_type"`
	Status         string   `bun:"status"`
	Description    string   `bun:"description"`
	AccountingDate int64    `bun:"accounting_date"`
	FiscalYearID   pulid.ID `bun:"fiscal_year_id"`
	FiscalPeriodID pulid.ID `bun:"fiscal_period_id"`
	EntryCount     int      `bun:"entry_count"`
	PostedAt       *int64   `bun:"posted_at,nullzero"`
	PostedByID     pulid.ID `bun:"posted_by_id,nullzero"`
	CreatedByID    pulid.ID `bun:"created_by_id"`
	UpdatedByID    pulid.ID `bun:"updated_by_id"`
}

type seedJournalEntry struct {
	bun.BaseModel `bun:"table:journal_entries"`

	ID              pulid.ID `bun:"id,pk"`
	BusinessUnitID  pulid.ID `bun:"business_unit_id,pk"`
	OrganizationID  pulid.ID `bun:"organization_id,pk"`
	BatchID         pulid.ID `bun:"batch_id"`
	FiscalYearID    pulid.ID `bun:"fiscal_year_id"`
	FiscalPeriodID  pulid.ID `bun:"fiscal_period_id"`
	EntryNumber     string   `bun:"entry_number"`
	EntryDate       int64    `bun:"entry_date"`
	EntryType       string   `bun:"entry_type"`
	AccountingDate  int64    `bun:"accounting_date"`
	Status          string   `bun:"status"`
	ReferenceNumber string   `bun:"reference_number"`
	ReferenceType   string   `bun:"reference_type"`
	ReferenceID     string   `bun:"reference_id"`
	Description     string   `bun:"description"`
	TotalDebit      int64    `bun:"total_debit"`
	TotalCredit     int64    `bun:"total_credit"`
	IsPosted        bool     `bun:"is_posted"`
	PostedAt        *int64   `bun:"posted_at,nullzero"`
	PostedByID      pulid.ID `bun:"posted_by_id,nullzero"`
	IsAutoGenerated bool     `bun:"is_auto_generated"`
	IsApproved      bool     `bun:"is_approved"`
	ApprovedAt      *int64   `bun:"approved_at,nullzero"`
	ApprovedByID    pulid.ID `bun:"approved_by_id,nullzero"`
	CreatedByID     pulid.ID `bun:"created_by_id"`
	UpdatedByID     pulid.ID `bun:"updated_by_id"`
}

type seedJournalEntryLine struct {
	bun.BaseModel `bun:"table:journal_entry_lines"`

	ID             pulid.ID `bun:"id,pk"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk"`
	OrganizationID pulid.ID `bun:"organization_id,pk"`
	JournalEntryID pulid.ID `bun:"journal_entry_id"`
	GLAccountID    pulid.ID `bun:"gl_account_id"`
	LineNumber     int16    `bun:"line_number"`
	Description    string   `bun:"description"`
	DebitAmount    int64    `bun:"debit_amount"`
	CreditAmount   int64    `bun:"credit_amount"`
	NetAmount      int64    `bun:"net_amount"`
}

type seedJournalSource struct {
	bun.BaseModel `bun:"table:journal_sources"`

	ID                   pulid.ID `bun:"id,pk"`
	BusinessUnitID       pulid.ID `bun:"business_unit_id,pk"`
	OrganizationID       pulid.ID `bun:"organization_id,pk"`
	SourceObjectType     string   `bun:"source_object_type"`
	SourceObjectID       string   `bun:"source_object_id"`
	SourceEventType      string   `bun:"source_event_type"`
	SourceDocumentNumber string   `bun:"source_document_number"`
	Status               string   `bun:"status"`
	IdempotencyKey       string   `bun:"idempotency_key"`
	JournalBatchID       pulid.ID `bun:"journal_batch_id"`
	JournalEntryID       pulid.ID `bun:"journal_entry_id"`
}

type seedSourceJournalLink struct {
	bun.BaseModel `bun:"table:source_journal_links"`

	ID              pulid.ID `bun:"id,pk"`
	BusinessUnitID  pulid.ID `bun:"business_unit_id,pk"`
	OrganizationID  pulid.ID `bun:"organization_id,pk"`
	JournalSourceID pulid.ID `bun:"journal_source_id"`
	JournalBatchID  pulid.ID `bun:"journal_batch_id"`
	JournalEntryID  pulid.ID `bun:"journal_entry_id"`
	LinkRole        string   `bun:"link_role"`
}

//nolint:funlen,gocognit // sequential journal fixture assembly reads clearest as one workflow
func (s *DriverPayLedgerSeed) postSeededSettlementJournals(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
	adminID pulid.ID,
	accounts map[string]pulid.ID,
) error {
	var settlements []*driversettlement.Settlement
	if err := tx.NewSelect().Model(&settlements).
		Relation("Lines").
		Where("dstl.organization_id = ?", orgID).
		Where("dstl.business_unit_id = ?", buID).
		Where("dstl.status IN (?)", bun.List([]driversettlement.Status{
			driversettlement.StatusPosted,
			driversettlement.StatusPaid,
		})).
		Where("dstl.posted_journal_batch_id IS NULL").
		Order("dstl.settlement_number ASC").
		Scan(ctx); err != nil {
		return fmt.Errorf("get unposted settlements: %w", err)
	}
	if len(settlements) == 0 {
		return nil
	}

	posted := 0
	for idx, settlement := range settlements {
		accountingDate := timeutils.NowUnix()
		if settlement.PostedAt != nil {
			accountingDate = *settlement.PostedAt
		}

		var period struct {
			ID           pulid.ID `bun:"id"`
			FiscalYearID pulid.ID `bun:"fiscal_year_id"`
		}
		err := tx.NewSelect().Table("fiscal_periods").Column("id", "fiscal_year_id").
			Where("organization_id = ?", orgID).
			Where("business_unit_id = ?", buID).
			Where("start_date <= ?", accountingDate).
			Where("end_date >= ?", accountingDate).
			Limit(1).
			Scan(ctx, &period)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return fmt.Errorf("get fiscal period: %w", err)
		}

		expenseAccountID := accounts[ledgerAccountDriverPayExpense]
		if settlement.Classification == driverpay.PayeeClassificationOwnerOperator {
			expenseAccountID = accounts[ledgerAccountPurchasedTrans]
		}
		codeAccounts := make(map[pulid.ID]pulid.ID)
		legs := driversettlementservice.BuildSettlementPostingLegs(
			settlement,
			&driversettlementservice.PostingAccounts{
				Expense:       expenseAccountID,
				Reimbursement: accounts[ledgerAccountDriverBenefits],
				Payable:       accounts[ledgerAccountSettlementPayable],
				Advance:       accounts[ledgerAccountDriverAdvances],
				Escrow:        accounts[ledgerAccountEscrowLiability],
			},
			codeAccounts,
		)
		if len(legs) == 0 {
			continue
		}

		description := "Driver settlement " + settlement.SettlementNumber
		batchID := pulid.MustNew("jb_")
		entryID := pulid.MustNew("je_")
		sourceID := pulid.MustNew("jsrc_")
		suffix := fmt.Sprintf("%04d", idx+1)

		var totalDebit, totalCredit int64
		lines := make([]*seedJournalEntryLine, 0, len(legs))
		for lineIdx, leg := range legs {
			totalDebit += leg.Debit
			totalCredit += leg.Credit
			lines = append(lines, &seedJournalEntryLine{
				ID:             pulid.MustNew("jel_"),
				BusinessUnitID: buID,
				OrganizationID: orgID,
				JournalEntryID: entryID,
				GLAccountID:    leg.AccountID,
				LineNumber:     int16(lineIdx + 1), //nolint:gosec // bounded by leg count
				Description:    description,
				DebitAmount:    leg.Debit,
				CreditAmount:   leg.Credit,
				NetAmount:      leg.Debit - leg.Credit,
			})
		}

		batch := &seedJournalBatch{
			ID:             batchID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			BatchNumber:    "SEED-JB-" + suffix,
			BatchType:      "System",
			Status:         "Posted",
			Description:    description,
			AccountingDate: accountingDate,
			FiscalYearID:   period.FiscalYearID,
			FiscalPeriodID: period.ID,
			EntryCount:     1,
			PostedAt:       &accountingDate,
			PostedByID:     adminID,
			CreatedByID:    adminID,
			UpdatedByID:    adminID,
		}
		if _, err = tx.NewInsert().Model(batch).Exec(ctx); err != nil {
			return fmt.Errorf("insert journal batch: %w", err)
		}
		if err = sc.TrackCreated(ctx, "journal_batches", batchID, s.Name()); err != nil {
			return fmt.Errorf("track journal batch: %w", err)
		}

		sourceEvent := tenant.JournalSourceEventDriverSettlementPosted.String()
		entry := &seedJournalEntry{
			ID:              entryID,
			BusinessUnitID:  buID,
			OrganizationID:  orgID,
			BatchID:         batchID,
			FiscalYearID:    period.FiscalYearID,
			FiscalPeriodID:  period.ID,
			EntryNumber:     "SEED-JE-" + suffix,
			EntryDate:       accountingDate,
			EntryType:       "Standard",
			AccountingDate:  accountingDate,
			Status:          "Posted",
			ReferenceNumber: settlement.SettlementNumber,
			ReferenceType:   sourceEvent,
			ReferenceID:     settlement.ID.String(),
			Description:     description,
			TotalDebit:      totalDebit,
			TotalCredit:     totalCredit,
			IsPosted:        true,
			PostedAt:        &accountingDate,
			PostedByID:      adminID,
			IsApproved:      true,
			ApprovedAt:      &accountingDate,
			ApprovedByID:    adminID,
			CreatedByID:     adminID,
			UpdatedByID:     adminID,
		}
		if _, err = tx.NewInsert().Model(entry).Exec(ctx); err != nil {
			return fmt.Errorf("insert journal entry: %w", err)
		}
		if err = sc.TrackCreated(ctx, "journal_entries", entryID, s.Name()); err != nil {
			return fmt.Errorf("track journal entry: %w", err)
		}

		if _, err = tx.NewInsert().Model(&lines).Exec(ctx); err != nil {
			return fmt.Errorf("insert journal entry lines: %w", err)
		}
		for _, line := range lines {
			if err = sc.TrackCreated(ctx, "journal_entry_lines", line.ID, s.Name()); err != nil {
				return fmt.Errorf("track journal entry line: %w", err)
			}
		}

		source := &seedJournalSource{
			ID:                   sourceID,
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			SourceObjectType:     "DriverSettlement",
			SourceObjectID:       settlement.ID.String(),
			SourceEventType:      sourceEvent,
			SourceDocumentNumber: settlement.SettlementNumber,
			Status:               "Posted",
			IdempotencyKey:       "driver-settlement-posted:" + settlement.ID.String(),
			JournalBatchID:       batchID,
			JournalEntryID:       entryID,
		}
		if _, err = tx.NewInsert().Model(source).Exec(ctx); err != nil {
			return fmt.Errorf("insert journal source: %w", err)
		}
		if err = sc.TrackCreated(ctx, "journal_sources", sourceID, s.Name()); err != nil {
			return fmt.Errorf("track journal source: %w", err)
		}

		link := &seedSourceJournalLink{
			ID:              sourceID,
			BusinessUnitID:  buID,
			OrganizationID:  orgID,
			JournalSourceID: sourceID,
			JournalBatchID:  batchID,
			JournalEntryID:  entryID,
			LinkRole:        "Primary",
		}
		if _, err = tx.NewInsert().Model(link).Exec(ctx); err != nil {
			return fmt.Errorf("insert source journal link: %w", err)
		}
		if err = sc.TrackCreated(ctx, "source_journal_links", link.ID, s.Name()); err != nil {
			return fmt.Errorf("track source journal link: %w", err)
		}

		for _, line := range lines {
			if err = s.applyBalances(ctx, tx, orgID, buID, period.FiscalYearID, period.ID,
				entryID, line); err != nil {
				return err
			}
		}

		if _, err = tx.NewUpdate().Model((*driversettlement.Settlement)(nil)).
			Set("posted_journal_batch_id = ?", batchID).
			Where("id = ?", settlement.ID).
			Where("organization_id = ?", orgID).
			Where("business_unit_id = ?", buID).
			Exec(ctx); err != nil {
			return fmt.Errorf("link settlement to journal batch: %w", err)
		}
		posted++
	}

	if posted > 0 {
		seedhelpers.LogSuccess(
			"Posted seeded settlements to the general ledger",
			fmt.Sprintf("- Created %d balanced journal batches with period balances for cost control", posted),
		)
	}
	return nil
}

func (s *DriverPayLedgerSeed) applyBalances(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID, fiscalYearID, fiscalPeriodID, entryID pulid.ID,
	line *seedJournalEntryLine,
) error {
	if _, err := tx.NewRaw(`
		INSERT INTO gl_account_balances_by_period AS gb (
			organization_id,
			business_unit_id,
			gl_account_id,
			fiscal_year_id,
			fiscal_period_id,
			period_debit_minor,
			period_credit_minor,
			net_change_minor,
			last_journal_entry_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (organization_id, business_unit_id, gl_account_id, fiscal_year_id, fiscal_period_id)
		DO UPDATE SET
			period_debit_minor = gb.period_debit_minor + EXCLUDED.period_debit_minor,
			period_credit_minor = gb.period_credit_minor + EXCLUDED.period_credit_minor,
			net_change_minor = gb.net_change_minor + EXCLUDED.net_change_minor,
			last_journal_entry_id = EXCLUDED.last_journal_entry_id,
			updated_at = extract(epoch from current_timestamp)::bigint
	`,
		orgID, buID, line.GLAccountID, fiscalYearID, fiscalPeriodID,
		line.DebitAmount, line.CreditAmount, line.NetAmount, entryID,
	).Exec(ctx); err != nil {
		return fmt.Errorf("upsert gl balance by period: %w", err)
	}

	if _, err := tx.NewUpdate().Table("gl_accounts").
		Set("current_balance = current_balance + ?", line.NetAmount).
		Set("debit_balance = debit_balance + ?", line.DebitAmount).
		Set("credit_balance = credit_balance + ?", line.CreditAmount).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Where("id = ?", line.GLAccountID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx); err != nil {
		return fmt.Errorf("update gl account running balance: %w", err)
	}

	return nil
}

func (s *DriverPayLedgerSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *DriverPayLedgerSeed) CanRollback() bool {
	return true
}
