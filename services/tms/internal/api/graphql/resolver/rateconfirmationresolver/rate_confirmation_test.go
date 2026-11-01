package rateconfirmationresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestRateConfirmationResolvers_CheckTheRateConfirmationPermission(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_").String()
	rateConID := pulid.MustNew("ratecon_").String()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"rateConfirmationsByMove", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).RateConfirmationsByMove(ctx, moveID)
			return err
		}},
		{
			"generateRateConfirmation",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).GenerateRateConfirmation(ctx, moveID)
				return err
			},
		},
		{"sendRateConfirmation", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SendRateConfirmation(ctx, rateConID)
			return err
		}},
		{
			"confirmRateConfirmation",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ConfirmRateConfirmation(
					ctx,
					rateConID,
					"Dana Ruiz",
				)
				return err
			},
		},
		{"voidRateConfirmation", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).VoidRateConfirmation(ctx, rateConID, "Wrong rate")
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceRateConfirmation,
				call.operation,
			)
		})
	}
}
