package manualjournalresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalcardservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
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
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestManualJournalCardListsWhatItBecameOnlyToSomeoneWhoMayReadJournalEntries(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		engine  permissionports.PermissionEngine
		entries int
	}{
		{name: "may read entries", engine: &mocks.AllowAllPermissionEngine{}, entries: 1},
		{
			name: "may not read entries",
			engine: &mocks.DenyResourcesPermissionEngine{
				Denied: []string{permission.ResourceJournalEntry.String()},
			},
			entries: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			journals := manualjournalportsmock.NewMockManualJournalRepository(t)
			sources := journalsourceportsmock.NewMockJournalSourceRepository(t)
			entries := journalentryportsmock.NewMockJournalEntryRepository(t)
			users := tenantportsmock.NewMockUserRepository(t)
			journal := &manualjournal.Request{
				ID:          pulid.MustNew("mjr_"),
				Status:      manualjournal.StatusPosted,
				CreatedByID: pulid.MustNew("usr_"),
			}
			posted := &journalentry.JournalEntry{
				ID:         pulid.MustNew("je_"),
				IsPosted:   true,
				PostedByID: journal.CreatedByID,
			}
			journals.EXPECT().GetByID(mock.Anything, mock.Anything).Return(journal, nil)
			sources.EXPECT().
				ListByObject(mock.Anything, mock.Anything).
				Return([]*journalsource.Source{{JournalEntryID: posted.ID}}, nil)
			entries.EXPECT().GetByID(mock.Anything, mock.Anything).Return(posted, nil)
			users.EXPECT().
				GetByIDs(mock.Anything, mock.Anything).
				Return([]*tenant.User{{ID: journal.CreatedByID, Name: "Dana Ruiz"}}, nil)

			resolver := &QueryResolver{&Deps{
				Core: &resolvercore.Core{L: zap.NewNop(), PermissionEngine: tc.engine},
				JournalCards: journalcardservice.New(journalcardservice.Params{
					Logger:    zap.NewNop(),
					Entries:   entries,
					Journals:  journals,
					Sources:   sources,
					Reversals: journalreversalportsmock.NewMockJournalReversalRepository(t),
					Accounts:  glaccountportsmock.NewMockGLAccountRepository(t),
					Periods:   fiscalperiodportsmock.NewMockFiscalPeriodRepository(t),
					Controls:  tenantportsmock.NewMockAccountingControlRepository(t),
					Users:     users,
					Customers: customerportsmock.NewMockCustomerRepository(t),
					Locations: locationportsmock.NewMockLocationRepository(t),
					Ledger: journalentryservice.New(journalentryservice.Params{
						Logger:     zap.NewNop(),
						EntryRepo:  entries,
						SourceRepo: sources,
					}),
				}),
			}}
			ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
				PrincipalType:  string(permissionports.PrincipalTypeUser),
				PrincipalID:    pulid.MustNew("usr_"),
				UserID:         pulid.MustNew("usr_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
			})

			card, err := resolver.ManualJournalCard(ctx, journal.ID.String())
			require.NoError(t, err)

			assert.Len(t, card.Entries, tc.entries)
			require.NotNil(t, card.PostedBy)
			assert.Equal(t, "Dana Ruiz", card.PostedBy.Name)
		})
	}
}
