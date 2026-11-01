package manualjournalresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManualJournalMutations_CheckTheManualJournalPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("mjr_").String()
	input := gqlmodel.ManualJournalDraftInput{
		Description:  "Accrue October rent",
		CurrencyCode: "USD",
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createManualJournalDraft",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateManualJournalDraft(ctx, input)
				return err
			},
		},
		{
			"updateManualJournalDraft",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateManualJournalDraft(ctx, id, input)
				return err
			},
		},
		{"submitManualJournal", permission.OpSubmit, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SubmitManualJournal(ctx, id)
			return err
		}},
		{"approveManualJournal", permission.OpApprove, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ApproveManualJournal(ctx, id)
			return err
		}},
		{"postManualJournal", permission.OpApprove, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PostManualJournal(ctx, id)
			return err
		}},
		{"rejectManualJournal", permission.OpReject, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).RejectManualJournal(ctx, id, "Wrong account")
			return err
		}},
		{"cancelManualJournal", permission.OpCancel, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CancelManualJournal(ctx, id, "Duplicate")
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			resolvertest.RequireRefused(
				t,
				call.run(ctx, deps),
				engine,
				permission.ResourceManualJournal,
				call.operation,
			)
		})
	}
}

func TestManualJournalLinesFromInput_MapsEachLineAndLeavesABlankAccountToTheService(t *testing.T) {
	t.Parallel()

	account := pulid.MustNew("gla_")
	customer := pulid.MustNew("cus_")
	customerRaw := customer.String()
	empty := ""

	lines, err := manualJournalLinesFromInput([]*gqlmodel.ManualJournalDraftLineInput{
		{
			GlAccountID: account.String(),
			Description: "Rent expense",
			DebitAmount: 250000,
			CustomerID:  &customerRaw,
			LocationID:  &empty,
		},
		{Description: "Accrued rent", CreditAmount: 250000},
	})

	require.NoError(t, err)
	require.Len(t, lines, 2)
	assert.Equal(t, account, lines[0].GLAccountID)
	assert.Equal(t, "Rent expense", lines[0].Description)
	assert.Equal(t, int64(250000), lines[0].DebitAmount)
	assert.Equal(t, customer, lines[0].CustomerID)
	assert.True(t, lines[0].LocationID.IsNil())
	assert.True(t, lines[1].GLAccountID.IsNil())
	assert.Equal(t, int64(250000), lines[1].CreditAmount)
}

func TestManualJournalLinesFromInput_NamesTheLineOfAMalformedID(t *testing.T) {
	t.Parallel()

	_, err := manualJournalLinesFromInput([]*gqlmodel.ManualJournalDraftLineInput{
		{GlAccountID: pulid.MustNew("gla_").String()},
		{GlAccountID: "not-an-id"},
	})

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, "lines[1].glAccountId", validationErr.Field)
}

func TestManualJournalLineGlAccount_ReadsTheAccountLoadedWithTheLine(t *testing.T) {
	t.Parallel()

	account := &glaccount.GLAccount{ID: pulid.MustNew("gla_"), AccountCode: "6100", Name: "Rent"}
	resolver := &ManualJournalLineResolver{}

	got, err := resolver.GlAccount(t.Context(), &manualjournal.Line{
		GLAccountID: account.ID,
		GLAccount:   account,
	})
	require.NoError(t, err)
	assert.Equal(t, &gqlmodel.JournalEntryLineAccount{
		ID:          account.ID.String(),
		AccountCode: "6100",
		Name:        "Rent",
	}, got)

	none, err := resolver.GlAccount(t.Context(), &manualjournal.Line{})
	require.NoError(t, err)
	assert.Nil(t, none)
}
