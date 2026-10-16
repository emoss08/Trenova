package journalentryresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalcardservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/glaccountportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalreversalportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalsourceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/locationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/manualjournalportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type cardMocks struct {
	entries *journalentryportsmock.MockJournalEntryRepository
	sources *journalsourceportsmock.MockJournalSourceRepository
	accts   *glaccountportsmock.MockGLAccountRepository
}

func cardResolver(
	t *testing.T,
	engine permissionports.PermissionEngine,
) (*QueryResolver, *cardMocks, context.Context) {
	t.Helper()

	m := &cardMocks{
		entries: journalentryportsmock.NewMockJournalEntryRepository(t),
		sources: journalsourceportsmock.NewMockJournalSourceRepository(t),
		accts:   glaccountportsmock.NewMockGLAccountRepository(t),
	}
	cards := journalcardservice.New(journalcardservice.Params{
		Logger:    zap.NewNop(),
		Entries:   m.entries,
		Journals:  manualjournalportsmock.NewMockManualJournalRepository(t),
		Sources:   m.sources,
		Reversals: journalreversalportsmock.NewMockJournalReversalRepository(t),
		Accounts:  m.accts,
		Periods:   fiscalperiodportsmock.NewMockFiscalPeriodRepository(t),
		Controls:  tenantportsmock.NewMockAccountingControlRepository(t),
		Users:     tenantportsmock.NewMockUserRepository(t),
		Customers: customerportsmock.NewMockCustomerRepository(t),
		Locations: locationportsmock.NewMockLocationRepository(t),
		Ledger: journalentryservice.New(journalentryservice.Params{
			Logger:     zap.NewNop(),
			EntryRepo:  m.entries,
			SourceRepo: m.sources,
		}),
	})
	deps := &Deps{
		Core:         &base.Core{L: zap.NewNop(), PermissionEngine: engine},
		JournalCards: cards,
	}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  string(permissionports.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	return &QueryResolver{deps}, m, ctx
}

func TestJournalEntryCardLeavesOutWhatTheReaderMayNotRead(t *testing.T) {
	t.Parallel()

	resolver, m, ctx := cardResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{
			permission.ResourceCustomer.String(),
			permission.ResourceLocation.String(),
			permission.ResourceJournalReversal.String(),
		},
	})
	receivable := &glaccount.GLAccount{
		ID:          pulid.MustNew("gla_"),
		AccountCode: "1200",
		Name:        "Accounts receivable",
		AccountType: &accounttype.AccountType{Category: accounttype.CategoryAsset},
	}
	entry := &journalentry.JournalEntry{
		ID:       pulid.MustNew("je_"),
		Status:   journalentry.StatusPosted,
		IsPosted: true,
		Lines: []*journalentry.JournalEntryLine{{
			LineNumber:  1,
			GLAccountID: receivable.ID,
			DebitAmount: 1000,
			CustomerID:  pulid.MustNew("cus_"),
			LocationID:  pulid.MustNew("loc_"),
		}},
	}
	m.entries.EXPECT().GetByID(mock.Anything, mock.Anything).Return(entry, nil)
	m.accts.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return([]*glaccount.GLAccount{receivable}, nil)
	m.sources.EXPECT().
		GetByJournalEntry(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("JournalSource not found"))

	card, err := resolver.JournalEntryCard(ctx, entry.ID.String())
	require.NoError(t, err)

	require.Len(t, card.Lines, 1)
	assert.Equal(t, "Accounts receivable", card.Lines[0].AccountName)
	assert.True(t, card.Lines[0].CustomerID.IsNil())
	assert.True(t, card.Lines[0].LocationID.IsNil())
	assert.Nil(t, card.Reversal)
}

func TestJournalEntryCardRefusesSomeoneWhoMayNotReadJournalEntries(t *testing.T) {
	t.Parallel()

	resolver, _, ctx := cardResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceJournalEntry.String()},
	})

	_, err := resolver.JournalEntryCard(ctx, pulid.MustNew("je_").String())
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestJournalEntryCardRefusesAnIDThatIsNotOne(t *testing.T) {
	t.Parallel()

	resolver, _, ctx := cardResolver(t, &mocks.AllowAllPermissionEngine{})

	_, err := resolver.JournalEntryCard(ctx, "not-an-id")
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestJournalCardFieldsReadNullWhenTheirValueIsUnknown(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &base.Core{L: zap.NewNop()}}
	line := &JournalCardLineResolver{deps}
	posting := &JournalCardPostingResolver{deps}
	source := &JournalSourceInfoResolver{deps}

	category, err := line.Category(t.Context(), &journalcardservice.Line{})
	require.NoError(t, err)
	assert.Nil(t, category)
	category, err = line.Category(
		t.Context(),
		&journalcardservice.Line{Category: accounttype.CategoryRevenue},
	)
	require.NoError(t, err)
	require.NotNil(t, category)
	assert.Equal(t, accounttype.CategoryRevenue, *category)

	date, err := posting.AccountingDate(
		t.Context(),
		&journalcardservice.Posting{Refusal: "The period is closed"},
	)
	require.NoError(t, err)
	assert.Nil(t, date)
	date, err = posting.AccountingDate(t.Context(), &journalcardservice.Posting{
		Period:         &fiscalperiod.FiscalPeriod{},
		AccountingDate: 1790812800,
	})
	require.NoError(t, err)
	require.NotNil(t, date)
	assert.Equal(t, 1790812800, *date)

	number, err := source.BatchNumber(t.Context(), &journalsource.Source{})
	require.NoError(t, err)
	assert.Nil(t, number)
	number, err = source.BatchNumber(t.Context(), &journalsource.Source{
		JournalBatch: &journalentry.JournalBatch{BatchNumber: "JB-2610-0031"},
	})
	require.NoError(t, err)
	require.NotNil(t, number)
	assert.Equal(t, "JB-2610-0031", *number)
}
