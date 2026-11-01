package webpushresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterMyPushSubscription_NamesTheMissingKeys(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).RegisterMyPushSubscription(
		ctx,
		gqlmodel.RegisterMyPushSubscriptionInput{Endpoint: "https://fcm.googleapis.com/fcm/send/x"},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"p256dh", "auth"}, fields)
}

func TestUnregisterMyPushSubscription_NeedsAnEndpoint(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).UnregisterMyPushSubscription(ctx, "")

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "endpoint", fieldErr.Field)
}

func TestWebPushConfig_RequiresASignedInCaller(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}

	_, err := (&QueryResolver{deps}).WebPushConfig(t.Context())

	var authnErr *errortypes.AuthenticationError
	require.ErrorAs(t, err, &authnErr)
}
