//go:build integration

package journalreversalservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalentryrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalpostingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalreversalrepository"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededRevOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}
type seededRevUser struct {
	ID pulid.ID `bun:"id"`
}
type seededRevAccount struct {
	ID pulid.ID `bun:"id"`
}

func TestJournalReversalPostingCreatesReversalEntryAndMarksOriginal(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	entryRepo := journalentryrepository.New(journalentryrepository.Params{DB: conn, Logger: logger})
	reversalRepo := journalreversalrepository.New(journalreversalrepository.Params{DB: conn, Logger: logger})
	postingRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededRevOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededRevUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	accounts := make([]seededRevAccount, 0, 2)
	require.NoError(t, db.NewSelect().Table("gl_accounts").Column("id").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Where("status = 'Active'").Order("account_code ASC").Limit(2).Scan(ctx, &accounts))
	require.Len(t, accounts, 2)

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.JournalReversalPolicy = tenant.JournalReversalPolicyNextOpenPeriod
	control.RequireManualJEApproval = false
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, Status: fiscalyear.StatusOpen, Year: 2027, Name: "FY 2027", StartDate: time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), EndDate: time.Date(2027, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(), IsCurrent: true, AllowAdjustingEntries: true})
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, FiscalYearID: fy.ID, PeriodNumber: 1, PeriodType: fiscalperiod.PeriodTypeMonth, Status: fiscalperiod.StatusOpen, Name: "January 2027", StartDate: time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), EndDate: time.Date(2027, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(), AllowAdjustingEntries: true})
	require.NoError(t, err)
	now := time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC).Unix()
	originalEntryID := pulid.MustNew("je_")
	require.NoError(t, postingRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{BatchID: pulid.MustNew("jb_"), OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID, BatchNumber: "JB-ORIG", BatchType: "Manual", BatchStatus: "Posted", BatchDescription: "Original", FiscalYearID: fy.ID, FiscalPeriodID: period.ID, AccountingDate: now, PostedAt: &now, PostedByID: user.ID, CreatedByID: user.ID, UpdatedByID: user.ID, EntryID: originalEntryID, EntryNumber: "JE-ORIG", EntryType: "Standard", EntryStatus: "Posted", ReferenceNumber: "MJR-1", ReferenceType: "ManualJournalRequest", ReferenceID: pulid.MustNew("mjr_").String(), EntryDescription: "Original", TotalDebit: 1000, TotalCredit: 1000, IsPosted: true, IsAutoGenerated: false, RequiresApproval: false, IsApproved: true, ApprovedByID: user.ID, ApprovedAt: &now, Lines: []repositories.JournalPostingLine{{ID: pulid.MustNew("jel_"), GLAccountID: accounts[0].ID, LineNumber: 1, Description: "Debit", DebitAmount: 1000, NetAmount: 1000}, {ID: pulid.MustNew("jel_"), GLAccountID: accounts[1].ID, LineNumber: 2, Description: "Credit", CreditAmount: 1000, NetAmount: -1000}}}))

	svc := New(Params{Logger: logger, DB: conn, JournalEntryRepo: entryRepo, JournalReversalRepo: reversalRepo, JournalPostingRepo: postingRepo, AccountingRepo: accountingRepo, SequenceGenerator: generator, Validator: NewValidator(ValidatorParams{FiscalRepo: fiscalPeriodRepo}), AuditService: &mocks.NoopAuditService{}})
	tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}
	actor := testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID)

	created, err := svc.Create(ctx, &serviceports.CreateJournalReversalRequest{OriginalJournalEntryID: originalEntryID, RequestedAccountingDate: now, ReasonCode: "CORR", ReasonText: "reverse original", TenantInfo: tenantInfo}, actor)
	require.NoError(t, err)
	require.Equal(t, journalreversal.StatusApproved, created.Status)

	posted, err := svc.Post(ctx, &serviceports.GetJournalReversalRequest{ReversalID: created.ID, TenantInfo: tenantInfo}, actor)
	require.NoError(t, err)
	require.Equal(t, journalreversal.StatusPosted, posted.Status)
	require.True(t, posted.ReversalJournalEntryID.IsNotNil())

	var reversalEntry struct {
		IsReversal   bool     `bun:"is_reversal"`
		ReversalOfID pulid.ID `bun:"reversal_of_id"`
		EntryType    string   `bun:"entry_type"`
	}
	require.NoError(t, db.NewSelect().Table("journal_entries").Column("is_reversal", "reversal_of_id", "entry_type").Where("id = ?", posted.ReversalJournalEntryID).Limit(1).Scan(ctx, &reversalEntry))
	assert.True(t, reversalEntry.IsReversal)
	assert.Equal(t, originalEntryID, reversalEntry.ReversalOfID)
	assert.Equal(t, "Reversal", reversalEntry.EntryType)

	var original struct {
		Status       string   `bun:"status"`
		ReversedByID pulid.ID `bun:"reversed_by_id"`
	}
	require.NoError(t, db.NewSelect().Table("journal_entries").Column("status", "reversed_by_id").Where("id = ?", originalEntryID).Limit(1).Scan(ctx, &original))
	assert.Equal(t, "Reversed", original.Status)
	assert.Equal(t, posted.ReversalJournalEntryID, original.ReversedByID)

	var debitBalance struct {
		NetChangeMinor    int64 `bun:"net_change_minor"`
		PeriodDebitMinor  int64 `bun:"period_debit_minor"`
		PeriodCreditMinor int64 `bun:"period_credit_minor"`
	}
	require.NoError(t, db.NewSelect().Table("gl_account_balances_by_period").Column("net_change_minor", "period_debit_minor", "period_credit_minor").Where("gl_account_id = ?", accounts[0].ID).Where("fiscal_period_id = ?", period.ID).Limit(1).Scan(ctx, &debitBalance))
	assert.Equal(t, int64(0), debitBalance.NetChangeMinor)
	assert.Equal(t, int64(1000), debitBalance.PeriodDebitMinor)
	assert.Equal(t, int64(1000), debitBalance.PeriodCreditMinor)
}
