//go:build integration

package manualjournalservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/glaccountrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalpostingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/manualjournalrepository"
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

type seededOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type seededUser struct {
	ID pulid.ID `bun:"id"`
}

type seededAccount struct {
	ID pulid.ID `bun:"id"`
}

func TestManualJournalPostingPersistsBatchEntryAndLines(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	manualRepo := manualjournalrepository.New(manualjournalrepository.Params{DB: conn, Logger: logger})
	journalRepo := journalpostingrepository.New(journalpostingrepository.Params{DB: conn, Logger: logger})
	accountingRepo := accountingcontrolrepository.New(accountingcontrolrepository.Params{DB: conn, Logger: logger})
	fiscalYearRepo := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: logger})
	fiscalPeriodRepo := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	glRepo := glaccountrepository.New(glaccountrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger})

	var org seededOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var user seededUser
	require.NoError(t, db.NewSelect().Table("users").Column("id").Where("current_organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Limit(1).Scan(ctx, &user))
	accounts := make([]seededAccount, 0, 2)
	require.NoError(t, db.NewSelect().Table("gl_accounts").Column("id").Where("organization_id = ?", org.ID).Where("business_unit_id = ?", org.BusinessUnitID).Where("allow_manual_je = TRUE").Where("status = 'Active'").Order("account_code ASC").Limit(2).Scan(ctx, &accounts))
	require.Len(t, accounts, 2)

	control, err := accountingRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	control.ManualJournalEntryPolicy = tenant.ManualJournalEntryPolicyAllowAll
	control.RequireManualJEApproval = false
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{
		OrganizationID:        org.ID,
		BusinessUnitID:        org.BusinessUnitID,
		Status:                fiscalyear.StatusOpen,
		Year:                  2027,
		Name:                  "FY 2027",
		StartDate:             time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:               time.Date(2027, time.December, 31, 23, 59, 59, 0, time.UTC).Unix(),
		IsCurrent:             true,
		AllowAdjustingEntries: true,
	})
	require.NoError(t, err)

	period, err := fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{
		OrganizationID:        org.ID,
		BusinessUnitID:        org.BusinessUnitID,
		FiscalYearID:          fy.ID,
		PeriodNumber:          1,
		PeriodType:            fiscalperiod.PeriodTypeMonth,
		Status:                fiscalperiod.StatusOpen,
		Name:                  "January 2027",
		StartDate:             time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:               time.Date(2027, time.January, 31, 23, 59, 59, 0, time.UTC).Unix(),
		AllowAdjustingEntries: true,
	})
	require.NoError(t, err)

	svc := New(Params{
		Logger:         logger,
		DB:             conn,
		Repo:           manualRepo,
		JournalRepo:    journalRepo,
		AccountingRepo: accountingRepo,
		Generator:      generator,
		Validator:      NewValidator(ValidatorParams{FiscalRepo: fiscalPeriodRepo, GLAccountRepo: glRepo}),
		AuditService:   &mocks.NoopAuditService{},
	})

	tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID, UserID: user.ID}
	actor := testutil.NewSessionActor(user.ID, org.ID, org.BusinessUnitID)

	draft, err := svc.CreateDraft(ctx, &serviceports.CreateManualJournalRequest{
		Description:    "Month-end accrual",
		Reason:         "Accrue detention expense",
		AccountingDate: period.StartDate,
		Lines: []*serviceports.ManualJournalLineInput{
			{GLAccountID: accounts[0].ID, Description: "Debit accrued expense", DebitAmount: 2500},
			{GLAccountID: accounts[1].ID, Description: "Credit accrued liability", CreditAmount: 2500},
		},
		TenantInfo: tenantInfo,
	}, actor)
	require.NoError(t, err)
	require.Equal(t, manualjournal.StatusDraft, draft.Status)

	submitted, err := svc.Submit(ctx, &serviceports.GetManualJournalRequest{RequestID: draft.ID, TenantInfo: tenantInfo}, actor)
	require.NoError(t, err)
	require.Equal(t, manualjournal.StatusApproved, submitted.Status)

	posted, err := svc.Post(ctx, &serviceports.GetManualJournalRequest{RequestID: draft.ID, TenantInfo: tenantInfo}, actor)
	require.NoError(t, err)
	require.Equal(t, manualjournal.StatusPosted, posted.Status)
	require.True(t, posted.PostedBatchID.IsNotNil())

	batchCount, err := db.NewSelect().Table("journal_batches").Where("id = ?", posted.PostedBatchID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, batchCount)

	var entry struct {
		ID            pulid.ID `bun:"id"`
		BatchID       pulid.ID `bun:"batch_id"`
		ReferenceType string   `bun:"reference_type"`
		ReferenceID   string   `bun:"reference_id"`
		TotalDebit    int64    `bun:"total_debit"`
		TotalCredit   int64    `bun:"total_credit"`
		Status        string   `bun:"status"`
	}
	require.NoError(t, db.NewSelect().Table("journal_entries").Column("id", "batch_id", "reference_type", "reference_id", "total_debit", "total_credit", "status").Where("batch_id = ?", posted.PostedBatchID).Limit(1).Scan(ctx, &entry))
	assert.Equal(t, posted.PostedBatchID, entry.BatchID)
	assert.Equal(t, "ManualJournalRequest", entry.ReferenceType)
	assert.Equal(t, draft.ID.String(), entry.ReferenceID)
	assert.Equal(t, int64(2500), entry.TotalDebit)
	assert.Equal(t, int64(2500), entry.TotalCredit)
	assert.Equal(t, "Posted", entry.Status)

	lineCount, err := db.NewSelect().Table("journal_entry_lines").Where("journal_entry_id = ?", entry.ID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, lineCount)
}
