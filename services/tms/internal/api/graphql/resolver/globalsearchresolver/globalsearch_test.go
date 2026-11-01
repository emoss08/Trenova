package globalsearchresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGlobalSearch_SearchesAsTheCaller(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	result := &kernelports.GlobalSearchResult{Query: "SEED-1"}
	service := kernelportsmock.NewMockGlobalSearchService(t)
	service.EXPECT().
		Search(mock.Anything, mock.MatchedBy(func(req *kernelports.GlobalSearchRequest) bool {
			return req.Query == "SEED-1" &&
				req.Limit == defaultSearchLimit &&
				req.TenantInfo.OrgID == authCtx.OrganizationID &&
				req.TenantInfo.BuID == authCtx.BusinessUnitID &&
				req.Principal.UserID == authCtx.UserID &&
				req.Principal.ID == authCtx.PrincipalID &&
				assert.ObjectsAreEqual(
					[]search.EntityType{search.EntityTypeShipment, search.EntityTypeWorker},
					req.EntityTypes,
				)
		})).
		Return(result, nil)
	deps := &Deps{Core: &resolvercore.Core{}, GlobalSearchService: service}

	got, err := (&QueryResolver{deps}).GlobalSearch(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.GlobalSearchInput{
			Query:       "  SEED-1 ",
			EntityTypes: []string{"shipment", "worker", "shipment", " "},
		},
	)
	require.NoError(t, err)
	assert.Same(t, result, got)
}

func TestGlobalSearch_RefusesAnUnknownEntityType(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}

	_, err := (&QueryResolver{deps}).GlobalSearch(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		gqlmodel.GlobalSearchInput{Query: "x", EntityTypes: []string{"invoice"}},
	)

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "entityTypes", fieldErr.Field)
}

func TestGlobalSearchHit_RendersItsKindAndMetadata(t *testing.T) {
	t.Parallel()

	hit := &kernelports.GlobalSearchHit{
		EntityType: search.EntityTypeCustomer,
		Metadata:   map[string]string{"Balance": "$10.00"},
	}
	resolver := &GlobalSearchHitResolver{&Deps{}}

	kind, err := resolver.EntityType(t.Context(), hit)
	require.NoError(t, err)
	assert.Equal(t, "customer", kind)

	metadata, err := resolver.Metadata(t.Context(), hit)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"Balance": "$10.00"}, metadata)
}
