//go:build integration

package journalreversalservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalentryrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalreversalrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance/glbalancedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestJournalReversalPostingCreatesReversalEntryAndMarksOriginal(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	entryRepo := journalentryrepository.New(journalentryrepository.Params{DB: conn, Logger: logger})
	reversalRepo := journalreversalrepository.New(
		journalreversalrepository.Params{DB: conn, Logger: logger},
	)
	postingRepo := journalpostingrepository.New(
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
	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	generator := seqgen.NewGenerator(
		seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
	)

	seeded := accountingtest.DevelopmentTenant(t, ctx, db)
	ctx = seeded.Ctx
	orgID, buID, userID := seeded.Info.OrgID, seeded.Info.BuID, seeded.Info.UserID

	gla := glaccountdb.GLAccounts
	accounts, err := gla.Where(gla.Status.Eq(domaintypes.StatusActive)).
		Order(gla.AccountCode.Asc()).
		Limit(2).
		List(ctx, db)
	require.NoError(t, err)
	require.Len(t, accounts, 2)

	control, err := accountingRepo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	control.JournalReversalPolicy = accountingcontrol.JournalReversalPolicyNextOpenPeriod
	control.RequireManualJEApproval = false
	_, err = accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	fy, err := fiscalYearRepo.Create(
		ctx,
		&fiscalyear.FiscalYear{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         fiscalyear.StatusOpen,
			Year:           2027,
			Name:           "FY 2027",
			StartDate:      time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
			EndDate: time.Date(2027, time.December, 31, 23, 59, 59, 0, time.UTC).
				Unix(),
			IsCurrent:             true,
			AllowAdjustingEntries: true,
		},
	)
	require.NoError(t, err)
	period, err := fiscalPeriodRepo.Create(
		ctx,
		&fiscalperiod.FiscalPeriod{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			FiscalYearID:   fy.ID,
			PeriodNumber:   1,
			PeriodType:     fiscalperiod.PeriodTypeMonth,
			Status:         fiscalperiod.StatusOpen,
			Name:           "January 2027",
			StartDate:      time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(),
			EndDate: time.Date(2027, time.January, 31, 23, 59, 59, 0, time.UTC).
				Unix(),
			AllowAdjustingEntries: true,
		},
	)
	require.NoError(t, err)
	now := time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC).Unix()
	originalEntryID := pulid.MustNew("je_")
	require.NoError(
		t,
		postingRepo.CreatePosting(
			ctx,
			journalentryports.CreateJournalPostingParams{
				BatchID:          pulid.MustNew("jb_"),
				OrganizationID:   orgID,
				BusinessUnitID:   buID,
				BatchNumber:      "JB-ORIG",
				BatchType:        "Manual",
				BatchStatus:      "Posted",
				BatchDescription: "Original",
				FiscalYearID:     fy.ID,
				FiscalPeriodID:   period.ID,
				AccountingDate:   now,
				PostedAt:         &now,
				PostedByID:       userID,
				CreatedByID:      userID,
				UpdatedByID:      userID,
				EntryID:          originalEntryID,
				EntryNumber:      "JE-ORIG",
				EntryType:        "Standard",
				EntryStatus:      "Posted",
				ReferenceNumber:  "MJR-1",
				ReferenceType:    "ManualJournalRequest",
				ReferenceID:      pulid.MustNew("mjr_").String(),
				EntryDescription: "Original",
				TotalDebit:       1000,
				TotalCredit:      1000,
				IsPosted:         true,
				IsAutoGenerated:  false,
				RequiresApproval: false,
				IsApproved:       true,
				ApprovedByID:     userID,
				ApprovedAt:       &now,
				Lines: []journalentryports.JournalPostingLine{
					{
						ID:          pulid.MustNew("jel_"),
						GLAccountID: accounts[0].ID,
						LineNumber:  1,
						Description: "Debit",
						DebitAmount: 1000,
						NetAmount:   1000,
					},
					{
						ID:           pulid.MustNew("jel_"),
						GLAccountID:  accounts[1].ID,
						LineNumber:   2,
						Description:  "Credit",
						CreditAmount: 1000,
						NetAmount:    -1000,
					},
				},
			},
		),
	)

	svc := New(
		Params{
			Logger:              logger,
			DB:                  conn,
			JournalEntryRepo:    entryRepo,
			JournalReversalRepo: reversalRepo,
			JournalPostingRepo:  postingRepo,
			AccountingRepo:      accountingRepo,
			SequenceGenerator:   generator,
			Validator:           NewValidator(ValidatorParams{FiscalRepo: fiscalPeriodRepo}),
			AuditService:        &mocks.NoopAuditService{},
		},
	)
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	actor := testutil.NewSessionActor(userID, orgID, buID)

	created, err := svc.Create(
		ctx,
		&journalreversalports.CreateJournalReversalRequest{
			OriginalJournalEntryID:  originalEntryID,
			RequestedAccountingDate: now,
			ReasonCode:              "CORR",
			ReasonText:              "reverse original",
			TenantInfo:              tenantInfo,
		},
		actor,
	)
	require.NoError(t, err)
	require.Equal(t, journalreversal.StatusApproved, created.Status)

	posted, err := svc.Post(
		ctx,
		&journalreversalports.GetJournalReversalRequest{
			ReversalID: created.ID,
			TenantInfo: tenantInfo,
		},
		actor,
	)
	require.NoError(t, err)
	require.Equal(t, journalreversal.StatusPosted, posted.Status)
	require.True(t, posted.ReversalJournalEntryID.IsNotNil())

	je := journalentrydb.JournalEntries
	reversalEntry, err := je.Get(ctx, db, posted.ReversalJournalEntryID)
	require.NoError(t, err)
	assert.True(t, reversalEntry.IsReversal)
	assert.Equal(t, originalEntryID, reversalEntry.ReversalOfID)
	assert.Equal(t, journalentry.EntryTypeReversal, reversalEntry.EntryType)

	original, err := je.Get(ctx, db, originalEntryID)
	require.NoError(t, err)
	assert.Equal(t, journalentry.StatusReversed, original.Status)
	assert.Equal(t, posted.ReversalJournalEntryID, original.ReversedByID)

	pb := glbalancedb.GLAccountPeriodBalances
	debitBalance, err := pb.Where(
		pb.GLAccountID.Eq(accounts[0].ID),
		pb.FiscalPeriodID.Eq(period.ID),
	).First(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), debitBalance.NetChangeMinor)
	assert.Equal(t, int64(1000), debitBalance.PeriodDebitMinor)
	assert.Equal(t, int64(1000), debitBalance.PeriodCreditMinor)
}
