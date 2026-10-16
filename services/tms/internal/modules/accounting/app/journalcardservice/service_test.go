package journalcardservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalsourceports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/glaccountportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalreversalportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalsourceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/locationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/manualjournalportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	octStart = int64(1790812800)
	octEnd   = int64(1793491199)
	sepDate  = int64(1790726400)
	novStart = int64(1793491200)
)

var (
	orgID    = pulid.MustNew("org_")
	buID     = pulid.MustNew("bu_")
	tenantIn = pagination.TenantInfo{OrgID: orgID, BuID: buID}
)

type fixture struct {
	entries   *journalentryportsmock.MockJournalEntryRepository
	journals  *manualjournalportsmock.MockManualJournalRepository
	sources   *journalsourceportsmock.MockJournalSourceRepository
	reversals *journalreversalportsmock.MockJournalReversalRepository
	accounts  *glaccountportsmock.MockGLAccountRepository
	periods   *fiscalperiodportsmock.MockFiscalPeriodRepository
	controls  *tenantportsmock.MockAccountingControlRepository
	users     *tenantportsmock.MockUserRepository
	customers *customerportsmock.MockCustomerRepository
	locations *locationportsmock.MockLocationRepository
	service   *Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		entries:   journalentryportsmock.NewMockJournalEntryRepository(t),
		journals:  manualjournalportsmock.NewMockManualJournalRepository(t),
		sources:   journalsourceportsmock.NewMockJournalSourceRepository(t),
		reversals: journalreversalportsmock.NewMockJournalReversalRepository(t),
		accounts:  glaccountportsmock.NewMockGLAccountRepository(t),
		periods:   fiscalperiodportsmock.NewMockFiscalPeriodRepository(t),
		controls:  tenantportsmock.NewMockAccountingControlRepository(t),
		users:     tenantportsmock.NewMockUserRepository(t),
		customers: customerportsmock.NewMockCustomerRepository(t),
		locations: locationportsmock.NewMockLocationRepository(t),
	}
	f.service = New(Params{
		Logger:    zap.NewNop(),
		Entries:   f.entries,
		Journals:  f.journals,
		Sources:   f.sources,
		Reversals: f.reversals,
		Accounts:  f.accounts,
		Periods:   f.periods,
		Controls:  f.controls,
		Users:     f.users,
		Customers: f.customers,
		Locations: f.locations,
		Ledger: journalentryservice.New(journalentryservice.Params{
			Logger:     zap.NewNop(),
			EntryRepo:  f.entries,
			SourceRepo: f.sources,
		}),
	})
	return f
}

func account(code, name string, category accounttype.Category) *glaccount.GLAccount {
	return &glaccount.GLAccount{
		ID:          pulid.MustNew("gla_"),
		AccountCode: code,
		Name:        name,
		AccountType: &accounttype.AccountType{Category: category},
	}
}

func openPeriod(number int, start, end int64) *fiscalperiod.FiscalPeriod {
	return &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: pulid.MustNew("fy_"),
		PeriodNumber: number,
		Status:       fiscalperiod.StatusOpen,
		Name:         "October 2026",
		StartDate:    start,
		EndDate:      end,
	}
}

func (f *fixture) expectAccounts(accounts ...*glaccount.GLAccount) {
	ids := make([]pulid.ID, 0, len(accounts))
	for _, a := range accounts {
		ids = append(ids, a.ID)
	}
	f.accounts.EXPECT().
		GetByIDs(mock.Anything, glaccountports.GetGLAccountsByIDsRequest{
			TenantInfo:         tenantIn,
			GLAccountIDs:       ids,
			IncludeAccountType: true,
		}).
		Return(accounts, nil)
}

func (f *fixture) expectUsers(ids []pulid.ID, users ...*tenant.User) {
	f.users.EXPECT().
		GetByIDs(mock.Anything, tenantports.GetUsersByIDsRequest{TenantInfo: tenantIn, UserIDs: ids}).
		Return(users, nil)
}

func postedInvoiceEntry(t *testing.T, receivable, revenue *glaccount.GLAccount) *journalentry.JournalEntry {
	t.Helper()
	return &journalentry.JournalEntry{
		ID:             pulid.MustNew("je_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		FiscalPeriodID: pulid.MustNew("fp_"),
		EntryNumber:    "JE-2610-00418",
		Status:         journalentry.StatusPosted,
		IsPosted:       true,
		AccountingDate: octStart + 13*86400,
		TotalDebit:     361250,
		TotalCredit:    361250,
		CreatedByID:    pulid.MustNew("usr_"),
		PostedByID:     pulid.MustNew("usr_"),
		Lines: []*journalentry.JournalEntryLine{
			{
				LineNumber:  1,
				GLAccountID: receivable.ID,
				Description: "Invoice INV-2610-0042",
				DebitAmount: 361250,
				CustomerID:  pulid.MustNew("cus_"),
				LocationID:  pulid.MustNew("loc_"),
			},
			{
				LineNumber:   2,
				GLAccountID:  revenue.ID,
				Description:  "Linehaul",
				CreditAmount: 361250,
			},
		},
	}
}

func TestJournalEntryCardNamesEachLinesAccountItsSideAndWhoMovedIt(t *testing.T) {
	f := newFixture(t)
	receivable := account("1200", "Accounts receivable", accounttype.CategoryAsset)
	revenue := account("4000", "Freight revenue", accounttype.CategoryRevenue)
	entry := postedInvoiceEntry(t, receivable, revenue)
	period := openPeriod(10, octStart, octEnd)
	period.ID = entry.FiscalPeriodID
	source := &journalsource.Source{
		SourceObjectType:     journalsource.ObjectInvoice,
		SourceObjectID:       "inv_1",
		SourceDocumentNumber: "INV-2610-0042",
		JournalBatch:         &journalentry.JournalBatch{BatchNumber: "JB-2610-0031"},
	}
	acme := &customer.Customer{ID: entry.Lines[0].CustomerID, Name: "Acme Foods"}
	creator := &tenant.User{ID: entry.CreatedByID, Name: "Trenova"}
	poster := &tenant.User{ID: entry.PostedByID, Name: "Dana Ruiz"}

	f.entries.EXPECT().
		GetByID(mock.Anything, journalentryports.GetJournalEntryByIDRequest{ID: entry.ID, TenantInfo: tenantIn}).
		Return(entry, nil)
	f.expectAccounts(receivable, revenue)
	f.customers.EXPECT().
		GetByIDs(mock.Anything, customerports.GetCustomersByIDsRequest{
			TenantInfo:  tenantIn,
			CustomerIDs: []pulid.ID{acme.ID},
		}).
		Return([]*customer.Customer{acme}, nil)
	f.periods.EXPECT().
		GetByID(mock.Anything, fiscalperiodports.GetFiscalPeriodByIDRequest{ID: period.ID, TenantInfo: tenantIn}).
		Return(period, nil)
	f.sources.EXPECT().
		GetByJournalEntry(mock.Anything, journalsourceports.GetJournalSourceByEntryRequest{
			TenantInfo:     tenantIn,
			JournalEntryID: entry.ID,
		}).
		Return(source, nil)
	f.expectUsers([]pulid.ID{entry.CreatedByID, entry.PostedByID}, creator, poster)

	card, err := f.service.GetJournalEntryCard(t.Context(), &GetJournalEntryCardRequest{
		TenantInfo: tenantIn,
		EntryID:    entry.ID,
		Parties:    Parties{IncludeCustomers: true},
	})
	require.NoError(t, err)

	assert.Equal(t, []*Line{
		{
			LineNumber:   1,
			GLAccountID:  receivable.ID,
			AccountCode:  "1200",
			AccountName:  "Accounts receivable",
			Category:     accounttype.CategoryAsset,
			Description:  "Invoice INV-2610-0042",
			DebitAmount:  361250,
			CustomerID:   acme.ID,
			CustomerName: "Acme Foods",
		},
		{
			LineNumber:   2,
			GLAccountID:  revenue.ID,
			AccountCode:  "4000",
			AccountName:  "Freight revenue",
			Category:     accounttype.CategoryRevenue,
			Description:  "Linehaul",
			CreditAmount: 361250,
		},
	}, card.Lines)
	assert.Same(t, period, card.Period)
	assert.Same(t, source, card.Source)
	assert.Nil(t, card.Posting)
	assert.Same(t, creator, card.CreatedBy)
	assert.Same(t, poster, card.PostedBy)
	assert.Nil(t, card.ApprovedBy)
	assert.Nil(t, card.Reversal)
}

func TestJournalEntryCardKeepsCustomersAndLocationsFromSomeoneWhoMayNotReadThem(t *testing.T) {
	f := newFixture(t)
	receivable := account("1200", "Accounts receivable", accounttype.CategoryAsset)
	revenue := account("4000", "Freight revenue", accounttype.CategoryRevenue)
	entry := postedInvoiceEntry(t, receivable, revenue)
	entry.FiscalPeriodID = pulid.Nil
	entry.CreatedByID = pulid.Nil
	entry.PostedByID = pulid.Nil

	f.entries.EXPECT().GetByID(mock.Anything, mock.Anything).Return(entry, nil)
	f.expectAccounts(receivable, revenue)
	f.sources.EXPECT().
		GetByJournalEntry(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("JournalSource not found"))

	card, err := f.service.GetJournalEntryCard(t.Context(), &GetJournalEntryCardRequest{
		TenantInfo: tenantIn,
		EntryID:    entry.ID,
	})
	require.NoError(t, err)

	assert.True(t, card.Lines[0].CustomerID.IsNil())
	assert.Empty(t, card.Lines[0].CustomerName)
	assert.True(t, card.Lines[0].LocationID.IsNil())
	assert.Nil(t, card.Source)
	assert.Nil(t, card.Period)
}

func TestJournalEntryCardSaysWhereAnEntryWaitingOnApprovalWillPost(t *testing.T) {
	f := newFixture(t)
	expense := account("5100", "Owner-operator pay", accounttype.CategoryExpense)
	entry := &journalentry.JournalEntry{
		ID:               pulid.MustNew("je_"),
		Status:           journalentry.StatusPending,
		RequiresApproval: true,
		AccountingDate:   sepDate,
		Lines: []*journalentry.JournalEntryLine{
			{LineNumber: 1, GLAccountID: expense.ID, DebitAmount: 100},
		},
	}
	september := openPeriod(9, sepDate-29*86400, sepDate+86399)
	september.Status = fiscalperiod.StatusClosed
	october := openPeriod(10, octStart, octEnd)
	october.FiscalYearID = september.FiscalYearID

	f.entries.EXPECT().GetByID(mock.Anything, mock.Anything).Return(entry, nil)
	f.expectAccounts(expense)
	f.controls.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{
			ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyPostToNextOpen,
		}, nil)
	f.periods.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{
			OrgID: orgID,
			BuID:  buID,
			Date:  sepDate,
		}).
		Return(september, nil)
	f.periods.EXPECT().
		ListByFiscalYearID(mock.Anything, mock.Anything).
		Return([]*fiscalperiod.FiscalPeriod{september, october}, nil)
	f.sources.EXPECT().
		GetByJournalEntry(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("JournalSource not found"))

	card, err := f.service.GetJournalEntryCard(t.Context(), &GetJournalEntryCardRequest{
		TenantInfo: tenantIn,
		EntryID:    entry.ID,
	})
	require.NoError(t, err)

	require.NotNil(t, card.Posting)
	assert.Same(t, october, card.Posting.Period)
	assert.Equal(t, octStart, card.Posting.AccountingDate)
	assert.Empty(t, card.Posting.Refusal)
}

func TestManualJournalCardSaysWhyItCannotPostWhenThePolicyRefusesItsClosedPeriod(t *testing.T) {
	f := newFixture(t)
	journal := &manualjournal.Request{
		ID:             pulid.MustNew("mjr_"),
		Status:         manualjournal.StatusDraft,
		AccountingDate: sepDate,
		CreatedByID:    pulid.MustNew("usr_"),
	}
	september := openPeriod(9, sepDate-29*86400, sepDate+86399)
	september.Status = fiscalperiod.StatusClosed

	f.journals.EXPECT().
		GetByID(mock.Anything, manualjournalports.GetManualJournalByIDRequest{ID: journal.ID, TenantInfo: tenantIn}).
		Return(journal, nil)
	f.controls.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{
			ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyRequireReopen,
		}, nil)
	f.periods.EXPECT().GetPeriodByDate(mock.Anything, mock.Anything).Return(september, nil)
	f.expectUsers([]pulid.ID{journal.CreatedByID}, &tenant.User{ID: journal.CreatedByID, Name: "Dana Ruiz"})

	card, err := f.service.GetManualJournalCard(t.Context(), &GetManualJournalCardRequest{
		TenantInfo: tenantIn,
		JournalID:  journal.ID,
	})
	require.NoError(t, err)

	require.NotNil(t, card.Posting)
	assert.Nil(t, card.Posting.Period)
	assert.Equal(
		t,
		"The manual journal falls in a closed fiscal period; reopen the period first",
		card.Posting.Refusal,
	)
	assert.Empty(t, card.Lines)
	assert.Equal(t, "Dana Ruiz", card.CreatedBy.Name)
}

func TestManualJournalCardNamesWhoPostedItEvenToSomeoneWhoMayNotReadItsEntry(t *testing.T) {
	f := newFixture(t)
	journal := &manualjournal.Request{
		ID:           pulid.MustNew("mjr_"),
		Status:       manualjournal.StatusPosted,
		CreatedByID:  pulid.MustNew("usr_"),
		ApprovedByID: pulid.MustNew("usr_"),
	}
	postedAt := octStart + 86400
	posted := &journalentry.JournalEntry{
		ID:         pulid.MustNew("je_"),
		IsPosted:   true,
		PostedAt:   &postedAt,
		PostedByID: journal.ApprovedByID,
	}
	approver := &tenant.User{ID: journal.ApprovedByID, Name: "Priya Shah"}

	f.journals.EXPECT().GetByID(mock.Anything, mock.Anything).Return(journal, nil)
	f.sources.EXPECT().
		ListByObject(mock.Anything, journalsourceports.GetJournalSourceByObjectRequest{
			TenantInfo:       tenantIn,
			SourceObjectType: journalsource.ObjectManualJournalRequest,
			SourceObjectID:   journal.ID.String(),
		}).
		Return([]*journalsource.Source{{JournalEntryID: posted.ID}}, nil)
	f.entries.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil)
	f.expectUsers(
		[]pulid.ID{journal.CreatedByID, journal.ApprovedByID},
		&tenant.User{ID: journal.CreatedByID, Name: "Dana Ruiz"},
		approver,
	)

	card, err := f.service.GetManualJournalCard(t.Context(), &GetManualJournalCardRequest{
		TenantInfo: tenantIn,
		JournalID:  journal.ID,
	})
	require.NoError(t, err)

	assert.Nil(t, card.Posting)
	assert.Same(t, approver, card.PostedBy)
	assert.Same(t, approver, card.ApprovedBy)
	assert.Equal(t, &postedAt, card.PostedAt)
	assert.Empty(t, card.Entries)
}

func TestPickReversalPrefersTheOneThatStandsOverANewerRefusal(t *testing.T) {
	entry := &journalentry.JournalEntry{ID: pulid.MustNew("je_")}
	other := pulid.MustNew("je_")
	rejected := &journalreversal.Reversal{
		OriginalJournalEntryID: entry.ID,
		Status:                 journalreversal.StatusRejected,
	}
	pending := &journalreversal.Reversal{
		OriginalJournalEntryID: entry.ID,
		Status:                 journalreversal.StatusPendingApproval,
	}
	reversing := &journalreversal.Reversal{
		OriginalJournalEntryID: other,
		ReversalJournalEntryID: entry.ID,
		Status:                 journalreversal.StatusPosted,
	}

	assert.Same(t, pending, PickReversal(entry, []*journalreversal.Reversal{rejected, pending}))
	assert.Same(t, rejected, PickReversal(entry, []*journalreversal.Reversal{rejected}))
	assert.Nil(t, PickReversal(entry, []*journalreversal.Reversal{reversing}))

	reversal := &journalentry.JournalEntry{ID: entry.ID, IsReversal: true}
	assert.Same(t, reversing, PickReversal(reversal, []*journalreversal.Reversal{pending, reversing}))
}
