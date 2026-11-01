package trailerresolver

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestBulkUpdateTrailerStatus_ChecksTheTrailerUpdatePermission(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).BulkUpdateTrailerStatus(
		ctx,
		gqlmodel.BulkUpdateTrailerStatusInput{
			TrailerIds: []string{pulid.MustNew("tr_").String()},
			Status:     domaintypes.EquipmentStatusAvailable,
		},
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceTrailer, permission.OpUpdate)
}

func TestBulkUpdateTrailerStatus_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey
	ctx := resolvertest.AuthedContext(t, authCtx)

	_, err := (&MutationResolver{deps}).BulkUpdateTrailerStatus(
		ctx,
		gqlmodel.BulkUpdateTrailerStatusInput{
			TrailerIds: []string{pulid.MustNew("tr_").String()},
			Status:     domaintypes.EquipmentStatusAvailable,
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
}
