package gqlext

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/subscription"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/zap"
)

type featureGate struct {
	disabled []platformcatalog.FeatureKey
}

func (g featureGate) Enabled(_ context.Context, feature platformcatalog.FeatureKey) (bool, error) {
	return !slices.Contains(g.disabled, feature), nil
}

func readOnlyOperation(operation ast.Operation) *gqlgen.OperationContext {
	return &gqlgen.OperationContext{
		OperationName: "TestOperation",
		Operation:     &ast.OperationDefinition{Operation: operation},
	}
}

func readOnlyContext(tenant pagination.TenantInfo, status *gqlctx.ResponseStatus) context.Context {
	ctx := gqlctx.WithAuthContext(context.Background(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		UserID:         tenant.UserID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	})

	return gqlctx.WithResponseStatus(ctx, status)
}

func TestReadOnlyExtensionRefusesMutationsFromAReadOnlyOrganization(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	ext := NewReadOnlyExtension(ReadOnlyParams{
		Plans: plantest.Cloud(
			t,
			tenant,
			plantest.FreeDemo(t, tenant, subscription.StatusReadOnly),
		),
		Features: featureGate{},
		Logger:   zap.NewNop(),
	})
	status := gqlctx.NewResponseStatus()

	gqlErr := ext.MutateOperationContext(readOnlyContext(tenant, status), readOnlyOperation(ast.Mutation))

	require.NotNil(t, gqlErr)
	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](gqlErr)
	require.True(t, ok)
	assert.Equal(t, platformplan.ReasonSubscriptionReadOnly, restriction.Reason)
	code, overridden := status.Code()
	assert.True(t, overridden)
	assert.Equal(t, http.StatusForbidden, code)
}

func TestReadOnlyExtensionLetsQueriesAndWritableOrganizationsThrough(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	readOnly := NewReadOnlyExtension(ReadOnlyParams{
		Plans: plantest.Cloud(
			t,
			tenant,
			plantest.FreeDemo(t, tenant, subscription.StatusReadOnly),
		),
		Features: featureGate{},
		Logger:   zap.NewNop(),
	})
	assert.Nil(t, readOnly.MutateOperationContext(
		readOnlyContext(tenant, gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Query),
	))

	trialing := NewReadOnlyExtension(ReadOnlyParams{
		Plans: plantest.Cloud(
			t,
			tenant,
			plantest.FreeDemo(t, tenant, subscription.StatusTrialing),
		),
		Features: featureGate{},
		Logger:   zap.NewNop(),
	})
	assert.Nil(t, trialing.MutateOperationContext(
		readOnlyContext(tenant, gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Mutation),
	))
}

func TestReadOnlyExtensionIsInertOutsideCloudMode(t *testing.T) {
	t.Parallel()

	ext := NewReadOnlyExtension(ReadOnlyParams{
		Plans:    planservice.NewUnlimited(),
		Features: featureGate{},
		Logger:   zap.NewNop(),
	})

	assert.Nil(t, ext.MutateOperationContext(
		readOnlyContext(plantest.Tenant(), gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Mutation),
	))
	assert.Equal(t, readOnlyExtensionName, ext.ExtensionName())
	require.NoError(t, ext.Validate(nil))
}

func TestReadOnlyExtensionRefusesMutationsFromASupersededSecret(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	ext := NewReadOnlyExtension(ReadOnlyParams{Features: featureGate{}, Logger: zap.NewNop()})
	status := gqlctx.NewResponseStatus()
	ctx := authctx.WithSupersededSessionSecret(readOnlyContext(tenant, status))

	gqlErr := ext.MutateOperationContext(ctx, readOnlyOperation(ast.Mutation))

	require.NotNil(t, gqlErr)
	stale, ok := errors.AsType[*errortypes.SessionContextStaleError](gqlErr)
	require.True(t, ok)
	assert.Equal(t, tenant.OrgID, stale.OrganizationID)
	code, overridden := status.Code()
	assert.True(t, overridden)
	assert.Equal(t, http.StatusConflict, code)
	assert.Nil(t, ext.MutateOperationContext(ctx, readOnlyOperation(ast.Query)))
}

func apiKeyContext(tenant pagination.TenantInfo, status *gqlctx.ResponseStatus) context.Context {
	ctx := gqlctx.WithAuthContext(context.Background(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeAPIKey,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	})

	return gqlctx.WithResponseStatus(ctx, status)
}

func TestReadOnlyExtensionRefusesAPIKeyMutationsWhenTheInstanceLacksTheFeature(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	ext := NewReadOnlyExtension(ReadOnlyParams{
		Plans: planservice.NewUnlimited(),
		Features: featureGate{
			disabled: []platformcatalog.FeatureKey{platformcatalog.FeatureAPIKeys},
		},
		Logger: zap.NewNop(),
	})
	status := gqlctx.NewResponseStatus()

	gqlErr := ext.MutateOperationContext(
		apiKeyContext(tenant, status),
		readOnlyOperation(ast.Mutation),
	)

	require.NotNil(t, gqlErr)
	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](gqlErr)
	require.True(t, ok)
	assert.Equal(t, errortypes.PlanRestrictionReasonPlan, restriction.Reason)
	assert.Equal(t, string(platformcatalog.FeatureAPIKeys), restriction.Feature)
	code, overridden := status.Code()
	assert.True(t, overridden)
	assert.Equal(t, http.StatusForbidden, code)

	assert.Nil(t, ext.MutateOperationContext(
		readOnlyContext(tenant, gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Mutation),
	))
}
