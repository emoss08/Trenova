package pagefavoriteresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToggleMyPageFavorite_NamesTheMissingFields(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).ToggleMyPageFavorite(
		ctx,
		gqlmodel.ToggleMyPageFavoriteInput{},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"pageUrl", "pageTitle"}, fields)
}

func TestIsMyPageFavorite_NeedsAPageURL(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&QueryResolver{deps}).IsMyPageFavorite(ctx, "")

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "pageUrl", fieldErr.Field)
}

func TestMyPageFavorites_RequireASignedInCaller(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}

	_, err := (&QueryResolver{deps}).MyPageFavorites(t.Context())

	var authnErr *errortypes.AuthenticationError
	require.ErrorAs(t, err, &authnErr)
}
