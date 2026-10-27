//go:build integration

package journalsourcerepository

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/glaccountrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalreversalrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalsourceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetByObjectReturnsLatestJournalSource(t *testing.T) {
	db := accountingtest.NewDevelopmentDatabase(t)
	ctx := db.A.Ctx
	conn := db.Conn
	repo := New(Params{DB: conn, Logger: zap.NewNop()})
	postingRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	orgID, buID, userID, fiscalYearID, periodID, accountIDs := setupJournalSourceFixture(t, db)
	now := time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC).Unix()
	sourceObjectID := pulid.MustNew("inv_").String()

	postInvoiceEntry(t, ctx, postingRepo, &postedFixture{
		orgID: orgID, buID: buID, userID: userID, fiscalYearID: fiscalYearID,
		periodID: periodID, accountIDs: accountIDs, at: now, sourceObjectID: sourceObjectID,
		number: "SRC",
	})

	source, err := repo.GetByObject(
		ctx,
		journalsourceports.GetJournalSourceByObjectRequest{
			TenantInfo:       pagination.TenantInfo{OrgID: orgID, BuID: buID},
			SourceObjectType: "Invoice",
			SourceObjectID:   sourceObjectID,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, source)
	assert.Equal(t, "Invoice", source.SourceObjectType)
	assert.Equal(t, sourceObjectID, source.SourceObjectID)
	assert.Equal(t, "InvoicePosted", source.SourceEventType)
	assert.Equal(t, "Posted", source.Status)
	assert.True(t, source.JournalEntryID.IsNotNil())
}

func setupJournalSourceFixture(
	t *testing.T,
	db *accountingtest.Database,
) (pulid.ID, pulid.ID, pulid.ID, pulid.ID, pulid.ID, []pulid.ID) {
	t.Helper()

	ctx := db.A.Ctx
	conn := db.Conn
	logger := zap.NewNop()
	fiscalYearRepo := fiscalyearrepository.New(
		fiscalyearrepository.Params{DB: conn, Logger: logger},
	)
	fiscalPeriodRepo := fiscalperiodrepository.New(
		fiscalperiodrepository.Params{DB: conn, Logger: logger},
	)
	org := struct{ ID, BusinessUnitID pulid.ID }{db.A.Info.OrgID, db.A.Info.BuID}
	user := struct{ ID pulid.ID }{db.A.Info.UserID}
	accounts := accountingtest.ActiveAccounts(t, db, db.A, 2)

	fy, err := fiscalYearRepo.Create(
		ctx,
		&fiscalyear.FiscalYear{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
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
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
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
	return org.ID, org.BusinessUnitID, user.ID, fy.ID, period.ID, accounts
}

type postedFixture struct {
	orgID          pulid.ID
	buID           pulid.ID
	userID         pulid.ID
	fiscalYearID   pulid.ID
	periodID       pulid.ID
	accountIDs     []pulid.ID
	at             int64
	sourceObjectID string
	number         string
}

func postInvoiceEntry(
	t *testing.T,
	ctx context.Context,
	postingRepo journalentryports.JournalPostingRepository,
	f *postedFixture,
) pulid.ID {
	t.Helper()
	orgID, buID, userID := f.orgID, f.buID, f.userID
	fiscalYearID, periodID, accountIDs := f.fiscalYearID, f.periodID, f.accountIDs
	now, sourceObjectID := f.at, f.sourceObjectID
	entryID := pulid.MustNew("je_")
	require.NoError(
		t,
		postingRepo.CreatePosting(
			ctx,
			journalentryports.CreateJournalPostingParams{
				BatchID:              pulid.MustNew("jb_"),
				OrganizationID:       orgID,
				BusinessUnitID:       buID,
				BatchNumber:          "JB-" + f.number,
				BatchType:            "System",
				BatchStatus:          "Posted",
				BatchDescription:     "Source seed",
				FiscalYearID:         fiscalYearID,
				FiscalPeriodID:       periodID,
				AccountingDate:       now,
				PostedAt:             &now,
				PostedByID:           userID,
				CreatedByID:          userID,
				UpdatedByID:          userID,
				EntryID:              entryID,
				EntryNumber:          "JE-" + f.number,
				EntryType:            "Standard",
				EntryStatus:          "Posted",
				ReferenceNumber:      "INV-1",
				ReferenceType:        "InvoicePosted",
				ReferenceID:          sourceObjectID,
				EntryDescription:     "Source seed",
				TotalDebit:           1000,
				TotalCredit:          1000,
				IsPosted:             true,
				IsAutoGenerated:      true,
				RequiresApproval:     false,
				IsApproved:           true,
				ApprovedByID:         userID,
				ApprovedAt:           &now,
				SourceID:             pulid.MustNew("jsrc_"),
				SourceObjectType:     "Invoice",
				SourceObjectID:       sourceObjectID,
				SourceEventType:      "InvoicePosted",
				SourceStatus:         "Posted",
				SourceDocumentNumber: "INV-1",
				SourceIdempotencyKey: "invoice-posted:" + sourceObjectID,
				Lines: []journalentryports.JournalPostingLine{
					{
						ID:          pulid.MustNew("jel_"),
						GLAccountID: accountIDs[0],
						LineNumber:  1,
						Description: "Debit",
						DebitAmount: 1000,
						NetAmount:   1000,
					},
					{
						ID:           pulid.MustNew("jel_"),
						GLAccountID:  accountIDs[1],
						LineNumber:   2,
						Description:  "Credit",
						CreditAmount: 1000,
						NetAmount:    -1000,
					},
				},
			},
		),
	)

	return entryID
}

func TestJournalCardReadsAnEntrysSourceReversalsAndAccountTypes(t *testing.T) {
	db := accountingtest.NewDevelopmentDatabase(t)
	ctx := db.A.Ctx
	conn := db.Conn
	logger := zap.NewNop()
	repo := New(Params{DB: conn, Logger: logger})
	postingRepo := journalpostingrepository.New(
		journalpostingrepository.Params{DB: conn, Logger: logger},
	)
	reversalRepo := journalreversalrepository.New(
		journalreversalrepository.Params{DB: conn, Logger: logger},
	)
	accountRepo := glaccountrepository.New(glaccountrepository.Params{DB: conn, Logger: logger})
	orgID, buID, userID, fiscalYearID, periodID, accountIDs := setupJournalSourceFixture(t, db)
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	at := time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC).Unix()
	fixture := func(number string) *postedFixture {
		return &postedFixture{
			orgID:          orgID,
			buID:           buID,
			userID:         userID,
			fiscalYearID:   fiscalYearID,
			periodID:       periodID,
			accountIDs:     accountIDs,
			at:             at,
			sourceObjectID: pulid.MustNew("inv_").String(),
			number:         number,
		}
	}
	original := postInvoiceEntry(t, ctx, postingRepo, fixture("ORIG"))
	mirror := postInvoiceEntry(t, ctx, postingRepo, fixture("MIRROR"))
	unrelated := postInvoiceEntry(t, ctx, postingRepo, fixture("OTHER"))

	source, err := repo.GetByJournalEntry(ctx, journalsourceports.GetJournalSourceByEntryRequest{
		TenantInfo:     tenantInfo,
		JournalEntryID: original,
	})
	require.NoError(t, err)
	assert.Equal(t, original, source.JournalEntryID)
	assert.Equal(t, "Invoice", source.SourceObjectType)
	require.NotNil(t, source.JournalBatch)
	assert.Equal(t, "JB-ORIG", source.JournalBatch.BatchNumber)

	_, err = repo.GetByJournalEntry(ctx, journalsourceports.GetJournalSourceByEntryRequest{
		TenantInfo:     tenantInfo,
		JournalEntryID: pulid.MustNew("je_"),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	reversal := func(original, reversing pulid.ID, status journalreversal.Status, createdAt int64) {
		_, createErr := reversalRepo.Create(ctx, &journalreversal.Reversal{
			OrganizationID:          orgID,
			BusinessUnitID:          buID,
			OriginalJournalEntryID:  original,
			ReversalJournalEntryID:  reversing,
			Status:                  status,
			RequestedAccountingDate: at,
			ResolvedFiscalYearID:    fiscalYearID,
			ResolvedFiscalPeriodID:  periodID,
			ReasonCode:              "DUPLICATE",
			ReasonText:              "Billed twice",
			RequestedByID:           userID,
			CreatedAt:               createdAt,
			UpdatedAt:               createdAt,
		})
		require.NoError(t, createErr)
	}
	reversal(original, pulid.Nil, journalreversal.StatusRejected, at)
	reversal(original, mirror, journalreversal.StatusPosted, at+60)
	reversal(unrelated, pulid.Nil, journalreversal.StatusPendingApproval, at+120)

	forOriginal, err := reversalRepo.ListByJournalEntry(
		ctx,
		journalreversalports.ListJournalReversalsByEntryRequest{
			TenantInfo:     tenantInfo,
			JournalEntryID: original,
		},
	)
	require.NoError(t, err)
	require.Len(t, forOriginal, 2)
	assert.Equal(t, journalreversal.StatusPosted, forOriginal[0].Status)
	assert.Equal(t, journalreversal.StatusRejected, forOriginal[1].Status)

	forMirror, err := reversalRepo.ListByJournalEntry(
		ctx,
		journalreversalports.ListJournalReversalsByEntryRequest{
			TenantInfo:     tenantInfo,
			JournalEntryID: mirror,
		},
	)
	require.NoError(t, err)
	require.Len(t, forMirror, 1)
	assert.Equal(t, original, forMirror[0].OriginalJournalEntryID)

	accounts, err := accountRepo.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
		TenantInfo:         tenantInfo,
		GLAccountIDs:       accountIDs,
		IncludeAccountType: true,
	})
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	for _, account := range accounts {
		require.NotNil(t, account.AccountType)
		assert.True(t, account.AccountType.Category.IsValid())
	}

	bare, err := accountRepo.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
		TenantInfo:   tenantInfo,
		GLAccountIDs: accountIDs,
	})
	require.NoError(t, err)
	for _, account := range bare {
		assert.Nil(t, account.AccountType)
	}
}
