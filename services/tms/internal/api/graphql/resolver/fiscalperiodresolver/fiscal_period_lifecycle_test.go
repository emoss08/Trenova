package fiscalperiodresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiscalPeriodOperations_CheckTheFiscalPeriodPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("fp_").String()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"fiscalPeriodCloseBlockers",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).FiscalPeriodCloseBlockers(ctx, id)
				return err
			},
		},
		{
			"activateFiscalPeriod",
			permission.OpActivate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ActivateFiscalPeriod(ctx, id)
				return err
			},
		},
		{"closeFiscalPeriod", permission.OpClose, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CloseFiscalPeriod(ctx, id)
			return err
		}},
		{"lockFiscalPeriod", permission.OpLock, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).LockFiscalPeriod(ctx, id)
			return err
		}},
		{"unlockFiscalPeriod", permission.OpUnlock, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UnlockFiscalPeriod(ctx, id)
			return err
		}},
		{"reopenFiscalPeriod", permission.OpReopen, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ReopenFiscalPeriod(ctx, id, "Late vendor invoice")
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
				permission.ResourceFiscalPeriod,
				call.operation,
			)
		})
	}
}

func TestFiscalCloseBlockerCode_RendersTheErrorCode(t *testing.T) {
	t.Parallel()

	code, err := (&FiscalCloseBlockerResolver{}).Code(t.Context(), &fiscalclose.Blocker{
		Field: "journalEntries",
		Code:  errortypes.ErrInvalid,
	})

	require.NoError(t, err)
	assert.Equal(t, string(errortypes.ErrInvalid), code)
}
