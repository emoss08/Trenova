package journalreversalresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJournalReversalMutations_CheckTheJournalReversalPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("jrev_").String()
	input := gqlmodel.CreateJournalReversalInput{
		OriginalJournalEntryID:  pulid.MustNew("je_").String(),
		RequestedAccountingDate: 1792108800,
		ReasonCode:              "DUPLICATE",
		ReasonText:              "Billed twice",
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createJournalReversal", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateJournalReversal(ctx, input)
			return err
		}},
		{
			"approveJournalReversal",
			permission.OpApprove,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ApproveJournalReversal(ctx, id)
				return err
			},
		},
		{"postJournalReversal", permission.OpApprove, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PostJournalReversal(ctx, id)
			return err
		}},
		{"rejectJournalReversal", permission.OpReject, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).RejectJournalReversal(ctx, id, "Not a duplicate")
			return err
		}},
		{"cancelJournalReversal", permission.OpCancel, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CancelJournalReversal(ctx, id, "Raised in error")
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
				permission.ResourceJournalReversal,
				call.operation,
			)
		})
	}
}

func TestCreateJournalReversal_NamesABlankOriginalEntry(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).CreateJournalReversal(
		ctx,
		gqlmodel.CreateJournalReversalInput{
			RequestedAccountingDate: 1792108800,
			ReasonCode:              "DUPLICATE",
		},
	)

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, "originalJournalEntryId", validationErr.Field)
}
