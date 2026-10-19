package development

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
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
		slices.Concat(seedhelpers.BaseSeedIDs, []seedhelpers.SeedID{seedhelpers.SeedDriverPay})...,
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

			if err = s.configureAccountingControl(
				ctx,
				tx,
				org.ID,
				org.BusinessUnitID,
				accounts,
			); err != nil {
				return fmt.Errorf("configure accounting control: %w", err)
			}

			if err = s.configureCostCategories(ctx, tx, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("configure cost categories: %w", err)
			}

			if err = s.ensureRecurringEarnings(
				ctx,
				tx,
				sc,
				org.ID,
				org.BusinessUnitID,
				admin.ID,
			); err != nil {
				return fmt.Errorf("ensure recurring earnings: %w", err)
			}

			if err = s.postSeededSettlementJournals(
				ctx,
				tx,
				sc,
				org.ID,
				org.BusinessUnitID,
				admin.ID,
				accounts,
			); err != nil {
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
	accounts, created, err := ensureGLAccounts(ctx, tx, sc, &seedGLAccountsRequest{
		seedName: s.Name(),
		orgID:    orgID,
		buID:     buID,
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
		fmt.Sprintf(
			"- Created %d recurring earnings (per diem, bonuses, stipends, equipment rental)",
			len(rows),
		),
	)
	return nil
}

// -- Settlement journals --

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
		journal, err := postSeedJournal(ctx, tx, sc, &seedJournalRequest{
			seedName:       s.Name(),
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

		if _, err = tx.NewUpdate().Model((*driversettlement.Settlement)(nil)).
			Set("posted_journal_batch_id = ?", journal.batchID).
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
			fmt.Sprintf(
				"- Created %d balanced journal batches with period balances for cost control",
				posted,
			),
		)
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
