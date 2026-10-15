package detentionresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type locationsByID struct {
	locationports.LocationRepository

	locations []*location.Location
	asked     int
}

func (l *locationsByID) GetByIDs(
	_ context.Context,
	_ locationports.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	l.asked++
	return l.locations, nil
}

var stopTenant = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

func signedInWithLocations(t *testing.T, repo *locationsByID) context.Context {
	t.Helper()

	userID := pulid.MustNew("usr_")
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: stopTenant.OrgID,
		BusinessUnitID: stopTenant.BuID,
	})
	return loaders.NewLocationByIDLoaderFactory(
		loaders.LocationByIDLoaderFactoryParams{LocationRepo: repo},
	).Attach(ctx, stopTenant)
}

func locationResolver(granted bool) *DetentionOccurrenceResolver {
	return &DetentionOccurrenceResolver{&Deps{Core: &base.Core{
		PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
			permission.ResourceLocation.String() + "|" + string(permission.OpRead): granted,
		}},
	}}}
}

func TestDetentionOccurrenceLocation_ReadsTheStopsLocationForSomeoneWhoMayReadLocations(t *testing.T) {
	t.Parallel()

	stop := &location.Location{ID: pulid.MustNew("loc_"), Name: "Kroger Atlanta DC", City: "Forest Park"}
	repo := &locationsByID{locations: []*location.Location{stop}}

	got, err := locationResolver(true).Location(
		signedInWithLocations(t, repo),
		&gqlmodel.DetentionOccurrence{LocationID: stop.ID.String()},
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Forest Park", got.City)
}

func TestDetentionOccurrenceLocation_IsNullWithoutPermissionAndNeverAsks(t *testing.T) {
	t.Parallel()

	repo := &locationsByID{locations: []*location.Location{{ID: pulid.MustNew("loc_")}}}

	got, err := locationResolver(false).Location(
		signedInWithLocations(t, repo),
		&gqlmodel.DetentionOccurrence{LocationID: repo.locations[0].ID.String()},
	)

	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Zero(t, repo.asked)
}

func TestDetentionOccurrenceLocation_IsNullOnceTheLocationIsGone(t *testing.T) {
	t.Parallel()

	got, err := locationResolver(true).Location(
		signedInWithLocations(t, &locationsByID{}),
		&gqlmodel.DetentionOccurrence{LocationID: pulid.MustNew("loc_").String()},
	)

	require.NoError(t, err)
	assert.Nil(t, got)
}
