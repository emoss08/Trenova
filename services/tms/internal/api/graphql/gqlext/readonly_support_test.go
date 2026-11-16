package gqlext

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/zap"
)

func supportContext(scope supportaccess.Scope, status *gqlctx.ResponseStatus) context.Context {
	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)
	ctx := gqlctx.WithAuthContext(context.Background(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		UserID:         principal,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Support:        &authctx.Support{StaffEmail: "jane.doe@trenova.app", Scope: scope.String()},
	})
	return gqlctx.WithResponseStatus(ctx, status)
}

func TestReadOnlyExtension_AReadOnlySupportSessionRunsNoMutation(t *testing.T) {
	t.Parallel()

	ext := NewReadOnlyExtension(ReadOnlyParams{
		Plans:    planservice.NewUnlimited(),
		Features: featureGate{},
		Logger:   zap.NewNop(),
	})
	status := gqlctx.NewResponseStatus()

	gqlErr := ext.MutateOperationContext(
		supportContext(supportaccess.ScopeReadOnly, status),
		readOnlyOperation(ast.Mutation),
	)

	require.NotNil(t, gqlErr)
	refusal, ok := errors.AsType[*errortypes.AuthorizationError](gqlErr)
	require.True(t, ok)
	assert.Contains(t, refusal.Message, "read-only")
	code, overridden := status.Code()
	assert.True(t, overridden)
	assert.Equal(t, http.StatusForbidden, code)
}

func TestReadOnlyExtension_ASupportSessionStillReadsAndAFullOneStillWrites(t *testing.T) {
	t.Parallel()

	ext := NewReadOnlyExtension(ReadOnlyParams{
		Plans:    planservice.NewUnlimited(),
		Features: featureGate{},
		Logger:   zap.NewNop(),
	})

	assert.Nil(t, ext.MutateOperationContext(
		supportContext(supportaccess.ScopeReadOnly, gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Query),
	))
	assert.Nil(t, ext.MutateOperationContext(
		supportContext(supportaccess.ScopeFull, gqlctx.NewResponseStatus()),
		readOnlyOperation(ast.Mutation),
	), "a full grant's writes are judged by the permission engine")
}
