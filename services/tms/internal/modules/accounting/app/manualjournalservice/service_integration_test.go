//go:build integration

package manualjournalservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/glaccountrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/manualjournalrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance/glbalancedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestManualJournalPostingPersistsBatchEntryAndLines(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	manualRepo := manualjournalrepository.New(
		manualjournalrepository.Params{DB: conn, Logger: logger},
	)
	journalRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	accountingRepo := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	fiscalYearRepo := fiscalyearrepository.New(
		fiscalyearrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	glRepo := glaccountrepository.New(glaccountrepository.Params{DB: conn, Logger: logger})
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	seeded := accountingtest.DevelopmentTenant(t, ctx, db)
	ctx = seeded.Ctx
	orgID, buID, userID := seeded.Info.OrgID, seeded.Info.BuID, seeded.Info.UserID

	gla := glaccountdb.GLAccounts
	accounts, err := gla.Where(
		gla.AllowManualJE.Eq(true),
		gla.Status.Eq(domaintypes.StatusActive),
	).Order(gla.AccountCode.Asc()).Limit(2).List(ctx, db)
	require.NoError(t, err)
	require.Len(t, accounts, 2)

	control, err := accountingRepo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	control.ManualJournalEntryPolicy = tenant.ManualJournalEntryPolicyAllowAll
	control.RequireManualJEApproval = false
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	fy, err := fiscalYearRepo.Create(ctx, &fiscalyear.FiscalYear{
		OrganizationID:        orgID,
		BusinessUnitID:        buID,
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
		OrganizationID:        orgID,
		BusinessUnitID:        buID,
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
		Validator: NewValidator(
			ValidatorParams{FiscalRepo: fiscalPeriodRepo, GLAccountRepo: glRepo},
		),
		AuditService: &mocks.NoopAuditService{},
	})

	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	actor := testutil.NewSessionActor(userID, orgID, buID)

	draft, err := svc.CreateDraft(ctx, &manualjournalports.CreateManualJournalRequest{
		Description:    "Month-end accrual",
		Reason:         "Accrue detention expense",
		AccountingDate: period.StartDate,
		Lines: []*manualjournalports.ManualJournalLineInput{
			{GLAccountID: accounts[0].ID, Description: "Debit accrued expense", DebitAmount: 2500},
			{
				GLAccountID:  accounts[1].ID,
				Description:  "Credit accrued liability",
				CreditAmount: 2500,
			},
		},
		TenantInfo: tenantInfo,
	}, actor)
	require.NoError(t, err)
	require.Equal(t, manualjournal.StatusDraft, draft.Status)

	submitted, err := svc.Submit(
		ctx,
		&manualjournalports.GetManualJournalRequest{RequestID: draft.ID, TenantInfo: tenantInfo},
		actor,
	)
	require.NoError(t, err)
	require.Equal(t, manualjournal.StatusApproved, submitted.Status)

	posted, err := svc.Post(
		ctx,
		&manualjournalports.GetManualJournalRequest{RequestID: draft.ID, TenantInfo: tenantInfo},
		actor,
	)
	require.NoError(t, err)
	require.Equal(t, manualjournal.StatusPosted, posted.Status)
	require.True(t, posted.PostedBatchID.IsNotNil())

	batchCount, err := journalentrydb.JournalBatches.
		Where(journalentrydb.JournalBatches.ID.Eq(posted.PostedBatchID)).
		Count(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), batchCount)

	je := journalentrydb.JournalEntries
	entry, err := je.Where(je.BatchID.Eq(posted.PostedBatchID)).First(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, posted.PostedBatchID, entry.BatchID)
	assert.Equal(t, "ManualJournalRequest", entry.ReferenceType)
	assert.Equal(t, draft.ID.String(), entry.ReferenceID)
	assert.Equal(t, int64(2500), entry.TotalDebit)
	assert.Equal(t, int64(2500), entry.TotalCredit)
	assert.Equal(t, journalentry.StatusPosted, entry.Status)

	jel := journalentrydb.JournalEntryLines
	lineCount, err := jel.Where(jel.JournalEntryID.Eq(entry.ID)).Count(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), lineCount)

	js := journalsourcedb.Sources
	source, err := js.Where(js.SourceObjectID.Eq(draft.ID.String())).First(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, "ManualJournalRequest", source.SourceObjectType)
	assert.Equal(t, "ManualJournalPosted", source.SourceEventType)
	assert.Equal(t, "Posted", source.Status)

	pb := glbalancedb.GLAccountPeriodBalances
	balance, err := pb.Where(
		pb.GLAccountID.Eq(accounts[0].ID),
		pb.FiscalPeriodID.Eq(period.ID),
	).First(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2500), balance.PeriodDebitMinor)
	assert.Equal(t, int64(0), balance.PeriodCreditMinor)
	assert.Equal(t, int64(2500), balance.NetChangeMinor)
}
