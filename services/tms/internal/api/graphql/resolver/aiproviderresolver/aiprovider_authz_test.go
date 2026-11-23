package aiproviderresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestProviderSettings_AIUsageViewDoesNotOpenThem(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		permission.ResourceAIUsage.String() + "|" + string(permission.OpRead):   true,
		permission.ResourceAIUsage.String() + "|" + string(permission.OpExport): true,
	}}
	q := &QueryResolver{&Deps{Core: &resolvercore.Core{PermissionEngine: engine}}}
	userID := pulid.MustNew("usr_")
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	var authz *errortypes.AuthorizationError
	_, err := q.AiProviders(ctx, gqlmodel.DataTableConnectionInput{})
	require.ErrorAs(t, err, &authz)
	_, err = q.AiProvider(ctx, pulid.MustNew("aiprv_").String())
	require.ErrorAs(t, err, &authz)
}
