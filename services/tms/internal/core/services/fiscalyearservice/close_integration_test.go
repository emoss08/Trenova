//go:build integration

package fiscalyearservice

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fiscalcloseservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/customerledgerrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/glaccountrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/glbalancerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalentryrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalpostingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalsourcerepository"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// countingSequenceGenerator hands out a distinct number per call, which the
// close needs because it posts two entries in one transaction.
type countingSequenceGenerator struct {
	seqgen.Generator

	counter atomic.Int64
}

func (g *countingSequenceGenerator) GenerateJournalBatchNumber(
	_ context.Context,
	_, _ pulid.ID,
	_, _ string,
) (string, error) {
	return fmt.Sprintf("JB-TEST-%04d", g.counter.Add(1)), nil
}

func (g *countingSequenceGenerator) GenerateJournalEntryNumber(
	_ context.Context,
	_, _ pulid.ID,
	_, _ string,
) (string, error) {
	return fmt.Sprintf("JE-TEST-%04d", g.counter.Add(1)), nil
}

type closeFixture struct {
	conn          *postgres.Connection
	postingRepo   repositories.JournalPostingRepository
	svc           *Service
	balanceRepo   repositories.GLBalanceRepository
	periodRepo    repositories.FiscalPeriodRepository
	orgID         pulid.ID
	buID          pulid.ID
	userID        pulid.ID
	closingYear   *fiscalyear.FiscalYear
	nextYear      *fiscalyear.FiscalYear
	assetID       pulid.ID
	revenueID     pulid.ID
	retainedID    pulid.ID
	closedPeriod  *fiscalperiod.FiscalPeriod
	nextYrPeriod1 *fiscalperiod.FiscalPeriod
}

func (f closeFixture) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: f.orgID, BuID: f.buID}
}

// postToAccount books a balanced entry that moves one account, offsetting it
// against revenue so the ledger still foots.
func (f closeFixture) postToAccount(
	ctx context.Context,
	accountID pulid.ID,
	amount int64,
	suffix string,
) error {
	return f.postingRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{
		BatchID:          pulid.MustNew("jb_"),
		OrganizationID:   f.orgID,
		BusinessUnitID:   f.buID,
		BatchNumber:      "JB-" + suffix,
		BatchType:        "System",
		BatchStatus:      "Posted",
		BatchDescription: "Control account drift",
		FiscalYearID:     f.closingYear.ID,
		FiscalPeriodID:   f.closedPeriod.ID,
		AccountingDate:   f.closingYear.StartDate,
		CreatedByID:      f.userID,
		EntryID:          pulid.MustNew("je_"),
		EntryNumber:      "JE-" + suffix,
		EntryType:        journalentry.EntryTypeStandard.String(),
		EntryStatus:      "Posted",
		EntryDescription: "Control account drift",
		TotalDebit:       amount,
		TotalCredit:      amount,
		IsPosted:         true,
		Lines: []repositories.JournalPostingLine{
			{
				ID:          pulid.MustNew("jel_"),
				GLAccountID: accountID,
				LineNumber:  1,
				Description: "Control account",
				DebitAmount: amount,
				NetAmount:   amount,
			},
			{
				ID:           pulid.MustNew("jel_"),
				GLAccountID:  f.revenueID,
				LineNumber:   2,
				Description:  "Offset",
				CreditAmount: amount,
				NetAmount:    -amount,
			},
		},
	})
}

func (f closeFixture) ytdBalance(
	t *testing.T,
	ctx context.Context,
	fiscalYearID, accountID pulid.ID,
) int64 {
	t.Helper()

	balances, err := f.balanceRepo.ListYearToDateBalances(
		ctx,
		repositories.ListYearToDateBalancesRequest{
			TenantInfo:   f.tenant(),
			FiscalYearID: fiscalYearID,
		},
	)
	require.NoError(t, err)

	for _, balance := range balances {
		if balance.GLAccountID == accountID {
			return balance.PeriodDebitMinor - balance.PeriodCreditMinor
		}
	}

	return 0
}

func TestCloseFiscalYearPostsClosingAndOpeningEntries(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := setupCloseFixture(t, ctx, db)

	closed, err := f.svc.Close(ctx, repositories.CloseFiscalYearRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	}, f.userID)
	require.NoError(t, err)
	assert.Equal(t, fiscalyear.StatusClosed, closed.Status)
	require.NotNil(t, closed.ClosedAt)

	// Revenue is emptied and the result lands in retained earnings.
	assert.Zero(t, f.ytdBalance(t, ctx, f.closingYear.ID, f.revenueID))
	assert.Equal(t, int64(-50000), f.ytdBalance(t, ctx, f.closingYear.ID, f.retainedID))
	assert.Equal(t, int64(50000), f.ytdBalance(t, ctx, f.closingYear.ID, f.assetID))

	// The next year opens from the position the old year ended in.
	assert.Equal(t, int64(50000), f.ytdBalance(t, ctx, f.nextYear.ID, f.assetID))
	assert.Equal(t, int64(-50000), f.ytdBalance(t, ctx, f.nextYear.ID, f.retainedID))
	assert.Zero(t, f.ytdBalance(t, ctx, f.nextYear.ID, f.revenueID))

	// The closing entry goes into a year-end adjusting period, so December keeps
	// showing December.
	periods, err := f.periodRepo.ListByFiscalYearID(ctx, repositories.ListByFiscalYearIDRequest{
		FiscalYearID: f.closingYear.ID,
		OrgID:        f.orgID,
		BuID:         f.buID,
	})
	require.NoError(t, err)

	var adjusting *fiscalperiod.FiscalPeriod
	for _, period := range periods {
		if period.IsAdjusting {
			adjusting = period
		}
	}
	require.NotNil(t, adjusting)
	assert.Equal(t, fiscalperiod.PeriodTypeAdjusting, adjusting.PeriodType)
	assert.Equal(t, fiscalperiod.StatusClosed, adjusting.Status)
	assert.Equal(t, 2, adjusting.PeriodNumber)

	entries := closeEntriesFor(t, ctx, f, f.closingYear.ID)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		assert.Equal(t, entry.TotalDebit, entry.TotalCredit)
		assert.True(t, entry.IsPosted)
	}
}

func TestCloseFiscalYearIsBlockedWithoutARetainedEarningsAccount(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := setupCloseFixture(t, ctx, db)
	_, err := f.conn.DB().NewUpdate().
		Table("accounting_controls").
		Set("default_retained_earnings_account_id = NULL").
		Where("organization_id = ?", f.orgID).
		Exec(ctx)
	require.NoError(t, err)

	result, err := f.svc.GetCloseBlockers(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	})
	require.NoError(t, err)
	assert.False(t, result.CanClose)

	_, err = f.svc.Close(ctx, repositories.CloseFiscalYearRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	}, f.userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retained earnings")
}

func TestReopenFiscalYearReversesTheClose(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := setupCloseFixture(t, ctx, db)

	_, err := f.svc.Close(ctx, repositories.CloseFiscalYearRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	}, f.userID)
	require.NoError(t, err)

	reopened, err := f.svc.Reopen(ctx, repositories.ReopenFiscalYearRequest{
		ID:           f.closingYear.ID,
		TenantInfo:   f.tenant(),
		ReopenReason: "Auditor found a misclassified accessorial",
	}, f.userID)
	require.NoError(t, err)
	assert.Equal(t, fiscalyear.StatusOpen, reopened.Status)
	assert.Nil(t, reopened.ClosedAt)
	require.NotNil(t, reopened.ReopenedAt)
	assert.Equal(t, "Auditor found a misclassified accessorial", reopened.ReopenReason)

	// The ledger is back where it was before the close.
	assert.Equal(t, int64(-50000), f.ytdBalance(t, ctx, f.closingYear.ID, f.revenueID))
	assert.Zero(t, f.ytdBalance(t, ctx, f.closingYear.ID, f.retainedID))
	assert.Zero(t, f.ytdBalance(t, ctx, f.nextYear.ID, f.assetID))
	assert.Zero(t, f.ytdBalance(t, ctx, f.nextYear.ID, f.retainedID))

	// Closing the year again posts a fresh pair rather than colliding with the
	// idempotency keys of the first close.
	_, err = f.svc.Close(ctx, repositories.CloseFiscalYearRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	}, f.userID)
	require.NoError(t, err)
	assert.Equal(t, int64(-50000), f.ytdBalance(t, ctx, f.nextYear.ID, f.retainedID))
}

func TestActivateRejectsAClosedFiscalYear(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := setupCloseFixture(t, ctx, db)
	_, err := f.svc.Close(ctx, repositories.CloseFiscalYearRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	}, f.userID)
	require.NoError(t, err)

	_, err = f.svc.Activate(ctx, repositories.ActivateFiscalYearRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	}, f.userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Reopen it first")
}

// Fiscal years generated after the adjusting-period change already carry an
// Inactive Period 13. The close must post into that one and seal it, not create
// a second adjusting period beside it.
func TestCloseFiscalYearReusesAnExistingAdjustingPeriod(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := setupCloseFixture(t, ctx, db)

	existing, err := f.periodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{
		OrganizationID:        f.orgID,
		BusinessUnitID:        f.buID,
		FiscalYearID:          f.closingYear.ID,
		PeriodNumber:          13,
		PeriodType:            fiscalperiod.PeriodTypeAdjusting,
		Status:                fiscalperiod.StatusInactive,
		Name:                  "Adjusting Period - FY 2031",
		StartDate:             f.closedPeriod.StartDate,
		EndDate:               f.closedPeriod.EndDate,
		IsAdjusting:           true,
		AllowAdjustingEntries: true,
	})
	require.NoError(t, err)

	_, err = f.svc.Close(ctx, repositories.CloseFiscalYearRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	}, f.userID)
	require.NoError(t, err)

	periods, err := f.periodRepo.ListByFiscalYearID(ctx, repositories.ListByFiscalYearIDRequest{
		FiscalYearID: f.closingYear.ID,
		OrgID:        f.orgID,
		BuID:         f.buID,
	})
	require.NoError(t, err)

	adjusting := make([]*fiscalperiod.FiscalPeriod, 0, 1)
	for _, period := range periods {
		if period.IsAdjusting {
			adjusting = append(adjusting, period)
		}
	}
	require.Len(t, adjusting, 1, "the close must not add a second adjusting period")
	assert.Equal(t, existing.ID, adjusting[0].ID)
	assert.Equal(t, fiscalperiod.StatusClosed, adjusting[0].Status,
		"the close seals the period it posted into")

	entries := closeEntriesFor(t, ctx, f, f.closingYear.ID)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		if entry.EntryType == journalentry.EntryTypeClosing {
			assert.Equal(t, existing.ID, entry.FiscalPeriodID)
		}
	}
}

// The GL carries one AR total; the customer detail behind it lives in the AR
// subledger, which has no fiscal year. The close proves the two still agree.
func TestClosePreviewReconcilesTheARControlAccount(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := setupCloseFixture(t, ctx, db)

	var arAccountID pulid.ID
	require.NoError(t, f.conn.DB().NewSelect().
		Table("accounting_controls").
		Column("default_ar_account_id").
		Where("organization_id = ?", f.orgID).
		Limit(1).
		Scan(ctx, &arAccountID))
	require.False(t, arAccountID.IsNil(), "the seeded accounting control names an AR account")

	plan, err := f.svc.GetClosePreview(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	})
	require.NoError(t, err)

	require.Len(t, plan.SubledgerChecks, 1)
	check := plan.SubledgerChecks[0]
	assert.Equal(t, "accounts_receivable", check.Key)
	assert.Equal(t, arAccountID, check.GLAccountID)

	// Nothing has been billed in this fixture, so both sides are empty and agree.
	assert.Zero(t, check.GLBalanceMinor)
	assert.Zero(t, check.SubledgerBalanceMinor)
	assert.True(t, check.Reconciled)

	// Reconciliation is off by default, so a difference would be reported without
	// stopping the close.
	assert.False(t, check.Enforced)
}

func TestClosePreviewReportsAnARDifference(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := setupCloseFixture(t, ctx, db)

	var arAccountID pulid.ID
	require.NoError(t, f.conn.DB().NewSelect().
		Table("accounting_controls").
		Column("default_ar_account_id").
		Where("organization_id = ?", f.orgID).
		Limit(1).
		Scan(ctx, &arAccountID))

	// 25,000 booked to the AR control account with nothing behind it in the
	// subledger: the detail no longer substantiates the total.
	require.NoError(t, f.postToAccount(ctx, arAccountID, 25000, "AR-DRIFT"))

	plan, err := f.svc.GetClosePreview(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	})
	require.NoError(t, err)

	require.Len(t, plan.SubledgerChecks, 1)
	check := plan.SubledgerChecks[0]
	assert.Equal(t, int64(25000), check.GLBalanceMinor)
	assert.Zero(t, check.SubledgerBalanceMinor)
	assert.Equal(t, int64(25000), check.DifferenceMinor)
	assert.False(t, check.Reconciled)

	// Unenforced, so it is reported but does not block.
	assert.True(t, plan.CanClose)

	_, err = f.conn.DB().NewUpdate().
		Table("accounting_controls").
		Set("require_reconciliation_to_close = TRUE").
		Set("reconciliation_mode = 'BlockPosting'").
		Where("organization_id = ?", f.orgID).
		Exec(ctx)
	require.NoError(t, err)

	plan, err = f.svc.GetClosePreview(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	})
	require.NoError(t, err)
	assert.False(t, plan.CanClose)
	require.NotEmpty(t, plan.Blockers)

	var reconciliationBlocked bool
	for _, blocker := range plan.Blockers {
		if blocker.Field == "reconciliation" {
			reconciliationBlocked = true
			assert.Contains(t, blocker.Message, "250.00")
		}
	}
	assert.True(t, reconciliationBlocked, "an enforced mismatch must block the close")

	_, err = f.svc.Close(ctx, repositories.CloseFiscalYearRequest{
		ID:         f.closingYear.ID,
		TenantInfo: f.tenant(),
	}, f.userID)
	require.Error(t, err)
}

func closeEntriesFor(
	t *testing.T,
	ctx context.Context,
	f closeFixture,
	fiscalYearID pulid.ID,
) []*journalentry.JournalEntry {
	t.Helper()

	entries := make([]*journalentry.JournalEntry, 0, 2)
	require.NoError(t, f.conn.DB().NewSelect().
		Model(&entries).
		Where("je.organization_id = ?", f.orgID).
		Where("je.business_unit_id = ?", f.buID).
		Where("je.reference_id = ?", fiscalYearID.String()).
		Where("je.entry_type IN (?, ?)", journalentry.EntryTypeClosing, journalentry.EntryTypeOpening).
		Where("je.reversed_by_id IS NULL").
		Scan(ctx))

	return entries
}

//nolint:funlen // one readable fixture beats several partial ones
func setupCloseFixture(t *testing.T, ctx context.Context, db *bun.DB) closeFixture {
	t.Helper()

	logger := zap.NewNop()
	conn := postgres.NewTestConnection(db)
	fyRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fpRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	balanceRepo := glbalancerepository.New(glbalancerepository.Params{DB: conn, Logger: logger})
	postingRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)

	// The seeded organization is the one that carries a chart of accounts and an
	// accounting control; a synthetic org would have neither.
	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, conn.DB().NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Limit(1).
		Scan(ctx, &org))
	orgID := org.ID
	buID := org.BusinessUnitID

	var userID pulid.ID
	require.NoError(t, conn.DB().NewSelect().
		Table("users").
		Column("id").
		Where("current_organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Limit(1).
		Scan(ctx, &userID))

	assetID := accountWithCategory(t, ctx, conn, orgID, buID, accounttype.CategoryAsset)
	revenueID := accountWithCategory(t, ctx, conn, orgID, buID, accounttype.CategoryRevenue)
	retainedID := accountWithCategory(t, ctx, conn, orgID, buID, accounttype.CategoryEquity)

	_, err := conn.DB().NewUpdate().
		Table("accounting_controls").
		Set("default_retained_earnings_account_id = ?", retainedID).
		Where("organization_id = ?", orgID).
		Exec(ctx)
	require.NoError(t, err)

	closingYear, err := fyRepo.Create(ctx, &fiscalyear.FiscalYear{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         fiscalyear.StatusOpen,
		Year:           2031,
		Name:           "FY 2031",
		StartDate:      time.Date(2031, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2031, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
	})
	require.NoError(t, err)

	nextYear, err := fyRepo.Create(ctx, &fiscalyear.FiscalYear{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         fiscalyear.StatusOpen,
		Year:           2032,
		Name:           "FY 2032",
		StartDate:      time.Date(2032, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2032, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:      true,
	})
	require.NoError(t, err)

	closingPeriod, err := fpRepo.Create(ctx, &fiscalperiod.FiscalPeriod{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		FiscalYearID:   closingYear.ID,
		PeriodNumber:   1,
		PeriodType:     fiscalperiod.PeriodTypeMonth,
		Status:         fiscalperiod.StatusOpen,
		Name:           "Period 1 - FY 2031",
		StartDate:      closingYear.StartDate,
		EndDate:        closingYear.EndDate,
	})
	require.NoError(t, err)

	nextPeriod, err := fpRepo.Create(ctx, &fiscalperiod.FiscalPeriod{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		FiscalYearID:   nextYear.ID,
		PeriodNumber:   1,
		PeriodType:     fiscalperiod.PeriodTypeMonth,
		Status:         fiscalperiod.StatusOpen,
		Name:           "Period 1 - FY 2032",
		StartDate:      nextYear.StartDate,
		EndDate:        nextYear.EndDate,
	})
	require.NoError(t, err)

	// One posted sale: 50,000 of revenue sitting in an asset account.
	require.NoError(t, postingRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{
		BatchID:          pulid.MustNew("jb_"),
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		BatchNumber:      "JB-SEED-0001",
		BatchType:        "System",
		BatchStatus:      "Posted",
		BatchDescription: "Seed activity",
		FiscalYearID:     closingYear.ID,
		FiscalPeriodID:   closingPeriod.ID,
		AccountingDate:   closingYear.StartDate,
		CreatedByID:      userID,
		EntryID:          pulid.MustNew("je_"),
		EntryNumber:      "JE-SEED-0001",
		EntryType:        journalentry.EntryTypeStandard.String(),
		EntryStatus:      "Posted",
		EntryDescription: "Seed activity",
		TotalDebit:       50000,
		TotalCredit:      50000,
		IsPosted:         true,
		Lines: []repositories.JournalPostingLine{
			{
				ID:          pulid.MustNew("jel_"),
				GLAccountID: assetID,
				LineNumber:  1,
				Description: "Receivable",
				DebitAmount: 50000,
				NetAmount:   50000,
			},
			{
				ID:           pulid.MustNew("jel_"),
				GLAccountID:  revenueID,
				LineNumber:   2,
				Description:  "Line haul revenue",
				CreditAmount: 50000,
				NetAmount:    -50000,
			},
		},
	}))

	_, err = fpRepo.Close(ctx, repositories.CloseFiscalPeriodRequest{
		ID:         closingPeriod.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		ClosedByID: userID,
		ClosedAt:   time.Now().Unix(),
	})
	require.NoError(t, err)

	svc := &Service{
		l:                logger,
		db:               conn,
		repo:             fyRepo,
		fiscalPeriodRepo: fpRepo,
		auditService:     &mocks.NoopAuditService{},
		closeService: fiscalcloseservice.New(fiscalcloseservice.Params{
			Logger:           logger,
			FiscalYearRepo:   fyRepo,
			FiscalPeriodRepo: fpRepo,
			GLBalanceRepo:    balanceRepo,
			GLAccountRepo: glaccountrepository.New(
				glaccountrepository.Params{DB: conn, Logger: logger},
			),
			AccountingRepo: accountingcontrolrepository.New(
				accountingcontrolrepository.Params{DB: conn, Logger: logger},
			),
			JournalPostRepo: postingRepo,
			CustomerLedger: customerledgerrepository.New(
				customerledgerrepository.Params{DB: conn, Logger: logger},
			),
			JournalEntryRepo: journalentryrepository.New(
				journalentryrepository.Params{DB: conn, Logger: logger},
			),
			SourceRepo: journalsourcerepository.New(
				journalsourcerepository.Params{DB: conn, Logger: logger},
			),
			Generator: &countingSequenceGenerator{},
		}),
	}

	return closeFixture{
		conn:          conn,
		postingRepo:   postingRepo,
		svc:           svc,
		balanceRepo:   balanceRepo,
		periodRepo:    fpRepo,
		orgID:         orgID,
		buID:          buID,
		userID:        userID,
		closingYear:   closingYear,
		nextYear:      nextYear,
		assetID:       assetID,
		revenueID:     revenueID,
		retainedID:    retainedID,
		closedPeriod:  closingPeriod,
		nextYrPeriod1: nextPeriod,
	}
}

func accountWithCategory(
	t *testing.T,
	ctx context.Context,
	conn *postgres.Connection,
	orgID, buID pulid.ID,
	category accounttype.Category,
) pulid.ID {
	t.Helper()

	var accountID pulid.ID
	require.NoError(t, conn.DB().NewSelect().
		TableExpr("gl_accounts AS gla").
		ColumnExpr("gla.id").
		Join("JOIN account_types AS at ON at.id = gla.account_type_id").
		Where("gla.organization_id = ?", orgID).
		Where("gla.business_unit_id = ?", buID).
		Where("gla.status = 'Active'").
		Where("at.category = ?", category).
		Order("gla.account_code ASC").
		Limit(1).
		Scan(ctx, &accountID))
	require.False(t, accountID.IsNil(), "no seeded %s account", category)

	return accountID
}
