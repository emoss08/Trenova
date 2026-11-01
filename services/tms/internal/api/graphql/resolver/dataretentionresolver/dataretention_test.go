package dataretentionresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func TestDataRetentionResolvers_CheckTheOrganizationPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"dataRetention", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DataRetention(ctx)
			return err
		}},
		{"updateDataRetention", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateDataRetention(
				ctx,
				gqlmodel.UpdateDataRetentionInput{AuditRetentionPeriod: 30},
			)
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
				permission.ResourceOrganization,
				call.operation,
			)
		})
	}
}
