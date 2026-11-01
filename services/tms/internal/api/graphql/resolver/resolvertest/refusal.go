package resolvertest

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RefusingEngine() *GrantingPermissionEngine {
	return &GrantingPermissionEngine{Granted: map[string]bool{}}
}

func UserAuthContext() *authctx.AuthContext {
	userID := pulid.MustNew("usr_")

	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func AuthedContext(t *testing.T, authCtx *authctx.AuthContext) context.Context {
	t.Helper()

	ctx := graphql.WithOperationContext(t.Context(), &graphql.OperationContext{})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{})

	return gqlctx.WithAuthContext(ctx, authCtx)
}

func RequireRefused(
	t *testing.T,
	err error,
	engine *GrantingPermissionEngine,
	resource permission.Resource,
	operation permission.Operation,
) {
	t.Helper()

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "the call must be refused: %v", err)
	require.Len(t, engine.Requests, 1)
	assert.Equal(t, resource.String(), engine.Requests[0].Resource)
	assert.Equal(t, operation, engine.Requests[0].Operation)
}
