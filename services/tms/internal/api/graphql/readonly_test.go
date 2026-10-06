package graphql

import (
	"context"
	"errors"
	"net/http"
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/zap"
)

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
		Plans:  plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusReadOnly)),
		Logger: zap.NewNop(),
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
		Plans:  plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusReadOnly)),
		Logger: zap.NewNop(),
	})
	assert.Nil(t, readOnly.MutateOperationContext(
		readOnlyContext(tenant, gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Query),
	))

	trialing := NewReadOnlyExtension(ReadOnlyParams{
		Plans:  plantest.Cloud(t, tenant, plantest.FreeDemo(t, tenant, subscription.StatusTrialing)),
		Logger: zap.NewNop(),
	})
	assert.Nil(t, trialing.MutateOperationContext(
		readOnlyContext(tenant, gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Mutation),
	))
}

func TestReadOnlyExtensionIsInertOutsideCloudMode(t *testing.T) {
	t.Parallel()

	ext := NewReadOnlyExtension(ReadOnlyParams{
		Plans:  planservice.NewUnlimited(),
		Logger: zap.NewNop(),
	})

	assert.Nil(t, ext.MutateOperationContext(
		readOnlyContext(plantest.Tenant(), gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Mutation),
	))
	assert.Equal(t, readOnlyExtensionName, ext.ExtensionName())
	require.NoError(t, ext.Validate(nil))
}
