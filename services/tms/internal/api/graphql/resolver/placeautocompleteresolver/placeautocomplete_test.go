package placeautocompleteresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlaceAutocomplete_PassesTheSearchInTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	token := "8f1c2a7e-3b9d-4c55-9f1a-2d6e0b7c4a11"
	result := &kernelports.AutocompleteLocationResult{Count: 1}
	service := kernelportsmock.NewMockAutoCompleteService(t)
	service.EXPECT().
		GetPlaceDetails(mock.Anything, &kernelports.AutoCompleteRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			Input:        "100 Main St",
			SessionToken: token,
		}).
		Return(result, nil)
	deps := &Deps{Core: &resolvercore.Core{}, AutoCompleteService: service}

	got, err := (&QueryResolver{deps}).PlaceAutocomplete(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.PlaceAutocompleteInput{Input: "100 Main St", SessionToken: &token},
	)
	require.NoError(t, err)
	assert.Same(t, result, got)
}

func TestPlaceAutocomplete_RequiresASignedInCaller(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}

	_, err := (&QueryResolver{deps}).PlaceAutocomplete(
		t.Context(),
		gqlmodel.PlaceAutocompleteInput{Input: "100 Main St"},
	)

	var authnErr *errortypes.AuthenticationError
	require.ErrorAs(t, err, &authnErr)
}
