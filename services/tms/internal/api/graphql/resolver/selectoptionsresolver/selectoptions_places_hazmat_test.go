package selectoptionsresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usstateservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/reference/app/dothazmatreferenceservice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/dothazmatreference"
	"github.com/emoss08/trenova/internal/testutil/mocks/dothazmatreferenceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/locationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/usstateportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSelectOptions_LocationByIDsCarriesItsAddressWithOneStateRead(t *testing.T) {
	t.Parallel()

	illinois := pulid.MustNew("us_")
	texas := pulid.MustNew("us_")
	chicago := pulid.MustNew("loc_")
	joliet := pulid.MustNew("loc_")
	dallas := pulid.MustNew("loc_")
	repo := locationportsmock.NewMockLocationRepository(t)
	repo.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return([]*location.Location{
			{
				ID:           chicago,
				Code:         "CHI01",
				Name:         "Chicago DC",
				AddressLine1: "100 W Lake St",
				City:         "Chicago",
				StateID:      illinois,
				PostalCode:   "60601",
			},
			{
				ID:           joliet,
				Code:         "JOL01",
				Name:         "Joliet Yard",
				AddressLine1: "2 Rail Rd",
				City:         "Joliet",
				StateID:      illinois,
				PostalCode:   "60431",
			},
			{
				ID:           dallas,
				Code:         "DAL01",
				Name:         "Dallas DC",
				AddressLine1: "5 Elm St",
				City:         "Dallas",
				StateID:      texas,
				PostalCode:   "75201",
			},
		}, nil).
		Once()
	states := usstateportsmock.NewMockUsStateRepository(t)
	states.EXPECT().
		GetByIDs(mock.Anything, []pulid.ID{illinois, texas}).
		Return([]*usstate.UsState{
			{ID: illinois, Name: "Illinois", Abbreviation: "IL"},
			{ID: texas, Name: "Texas", Abbreviation: "TX"},
		}, nil).
		Once()

	resolver := &QueryResolver{&Deps{
		Core: &resolvercore.Core{},
		LocationService: locationservice.New(locationservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
		UsStateService: usstateservice.New(usstateservice.Params{
			Logger: zap.NewNop(),
			Repo:   states,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceLocation,
		Ids:      []string{dallas.String(), chicago.String(), joliet.String()},
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 3)
	assert.Equal(t, "Dallas DC", result.Edges[0].Node.Label)
	assert.Equal(t, map[string]any{
		"code":              "DAL01",
		"addressLine1":      "5 Elm St",
		"city":              "Dallas",
		"stateAbbreviation": "TX",
		"postalCode":        "75201",
	}, result.Edges[0].Node.Meta)
	assert.Equal(t, "IL", result.Edges[1].Node.Meta["stateAbbreviation"])
	assert.Equal(t, "IL", result.Edges[2].Node.Meta["stateAbbreviation"])
}

func TestSelectOptions_LocationSearchUsesTheStateItLoaded(t *testing.T) {
	t.Parallel()

	repo := locationportsmock.NewMockLocationRepository(t)
	repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *locationports.LocationSelectOptionsRequest) bool {
			return req.SelectQueryRequest.Query == "chi"
		})).
		Return(&pagination.ListResult[*location.Location]{
			Items: []*location.Location{{
				ID:      pulid.MustNew("loc_"),
				Code:    "CHI01",
				Name:    "Chicago DC",
				City:    "Chicago",
				StateID: pulid.MustNew("us_"),
				State:   &usstate.UsState{Abbreviation: "IL"},
			}},
			Total:      1,
			TotalExact: true,
		}, nil).
		Once()

	resolver := &QueryResolver{&Deps{
		Core: &resolvercore.Core{},
		LocationService: locationservice.New(locationservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	query := "chi"
	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceLocation,
		Query:    &query,
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, "Chicago", result.Edges[0].Node.Meta["city"])
	assert.Equal(t, "IL", result.Edges[0].Node.Meta["stateAbbreviation"])
}

func dotHazmatResolver(
	repo *dothazmatreferenceportsmock.MockDotHazmatReferenceRepository,
	engine *resolvertest.RecordingPermissionEngine,
) *QueryResolver {
	return &QueryResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: engine},
		DotHazmatReferenceService: dothazmatreferenceservice.New(
			dothazmatreferenceservice.Params{Logger: zap.NewNop(), Repo: repo},
		),
	}}
}

func TestSelectOptions_DotHazmatReferenceSearchCarriesWhatTheHazmatFormCopies(t *testing.T) {
	t.Parallel()

	referenceID := pulid.MustNew("dhr_")
	repo := dothazmatreferenceportsmock.NewMockDotHazmatReferenceRepository(t)
	repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *pagination.SelectQueryRequest) bool {
			return req.Query == "1203" && req.Pagination.Limit == 10
		})).
		Return(&pagination.ListResult[*dothazmatreference.DotHazmatReference]{
			Items: []*dothazmatreference.DotHazmatReference{{
				ID:                 referenceID,
				UnNumber:           "1203",
				ProperShippingName: "Gasoline",
				HazardClass:        "3",
				PackingGroup:       "II",
				SpecialProvisions:  "144, 177",
				ErgGuide:           "128",
				Symbols:            "RQ",
			}},
			Total:      1,
			TotalExact: true,
		}, nil).
		Once()
	engine := &resolvertest.RecordingPermissionEngine{}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	query := "1203"
	limit := 10
	result, err := dotHazmatResolver(repo, engine).SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceDOTHazmatReference,
		Query:    &query,
		First:    &limit,
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	node := result.Edges[0].Node
	assert.Equal(t, referenceID.String(), node.ID)
	assert.Equal(t, "Gasoline", node.Label)
	require.NotNil(t, node.Description)
	assert.Equal(t, "UN1203", *node.Description)
	assert.Equal(t, map[string]any{
		"unNumber":           "1203",
		"properShippingName": "Gasoline",
		"hazardClass":        "3",
		"subsidiaryHazard":   "",
		"packingGroup":       "II",
		"specialProvisions":  "144, 177",
		"ergGuide":           "128",
		"symbols":            "RQ",
	}, node.Meta)
	assert.Nil(t, engine.Request)
}

func TestSelectOptions_DotHazmatReferenceReadsSelectedIDsInOneReadInTheirOrder(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("dhr_")
	second := pulid.MustNew("dhr_")
	gone := pulid.MustNew("dhr_")
	repo := dothazmatreferenceportsmock.NewMockDotHazmatReferenceRepository(t)
	repo.EXPECT().
		GetByIDs(mock.Anything, []pulid.ID{second, first, gone}).
		Return([]*dothazmatreference.DotHazmatReference{
			{ID: first, UnNumber: "1203", ProperShippingName: "Gasoline", HazardClass: "3"},
			{ID: second, UnNumber: "1993", ProperShippingName: "Diesel fuel", HazardClass: "3"},
		}, nil).
		Once()
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	result, err := dotHazmatResolver(repo, &resolvertest.RecordingPermissionEngine{}).
		SelectOptions(ctx, gqlmodel.SelectOptionsInput{
			Resource: gqlmodel.SelectOptionResourceDOTHazmatReference,
			Ids:      []string{second.String(), first.String(), gone.String()},
		})
	require.NoError(t, err)

	require.Len(t, result.Edges, 2)
	assert.Equal(t, "Diesel fuel", result.Edges[0].Node.Label)
	assert.Equal(t, "Gasoline", result.Edges[1].Node.Label)
}
