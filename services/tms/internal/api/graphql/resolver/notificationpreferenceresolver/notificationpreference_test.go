package notificationpreferenceresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestMyNotificationPreferences_RefuseAnAgent(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}
	agent := resolvertest.UserAuthContext()
	agent.PrincipalType = authctx.PrincipalTypeAgent
	ctx := resolvertest.AuthedContext(t, agent)

	_, err := (&QueryResolver{deps}).MyNotificationPreferences(ctx)
	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)

	_, err = (&MutationResolver{deps}).UpdateMyNotificationPreferences(
		ctx,
		gqlmodel.UpdateMyNotificationPreferencesInput{},
	)
	require.ErrorAs(t, err, &authzErr)
}

func TestMyNotificationPreferences_RequireASignedInCaller(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}

	_, err := (&QueryResolver{deps}).MyNotificationPreferences(t.Context())

	var authnErr *errortypes.AuthenticationError
	require.ErrorAs(t, err, &authnErr)
}
