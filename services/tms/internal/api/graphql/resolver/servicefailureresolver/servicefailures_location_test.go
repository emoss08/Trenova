package servicefailureresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingServiceFailures struct {
	servicefailureports.ServiceFailureService

	requests []*servicefailureports.ListServiceFailureConnectionRequest
}

func (s *capturingServiceFailures) ListConnection(
	_ context.Context,
	req *servicefailureports.ListServiceFailureConnectionRequest,
) (*pagination.CursorListResult[*servicefailure.ServiceFailure], error) {
	s.requests = append(s.requests, req)
	return &pagination.CursorListResult[*servicefailure.ServiceFailure]{}, nil
}

func listContext(t *testing.T) context.Context {
	t.Helper()

	ctx := graphql.WithOperationContext(t.Context(), &graphql.OperationContext{})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{})
	userID := pulid.MustNew("usr_")

	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func newListResolver(service servicefailureports.ServiceFailureService) *QueryResolver {
	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		permission.ResourceServiceFailure.String() + "|" + string(permission.OpRead): true,
	}}
	return &QueryResolver{&Deps{
		Core:              &base.Core{PermissionEngine: engine},
		ServiceFailureSvc: service,
	}}
}

func TestServiceFailures_LocationIDReachesTheListRequest(t *testing.T) {
	t.Parallel()

	service := &capturingServiceFailures{}
	locationID := pulid.MustNew("loc_")
	raw := locationID.String()
	first := 5

	_, err := newListResolver(service).ServiceFailures(
		listContext(t),
		gqlmodel.DataTableConnectionInput{First: &first},
		nil,
		&raw,
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	require.NotNil(t, service.requests[0].LocationID)
	assert.Equal(t, locationID, *service.requests[0].LocationID)
	assert.Nil(t, service.requests[0].ShipmentID)
}

func TestServiceFailures_NoLocationLeavesTheFilterOff(t *testing.T) {
	t.Parallel()

	service := &capturingServiceFailures{}
	empty := ""

	for _, locationID := range []*string{nil, &empty} {
		_, err := newListResolver(service).ServiceFailures(
			listContext(t),
			gqlmodel.DataTableConnectionInput{},
			nil,
			locationID,
		)
		require.NoError(t, err)
	}
	require.Len(t, service.requests, 2)
	assert.Nil(t, service.requests[0].LocationID)
	assert.Nil(t, service.requests[1].LocationID)
}

func TestServiceFailures_RejectsAMalformedLocationID(t *testing.T) {
	t.Parallel()

	service := &capturingServiceFailures{}
	raw := "dallas-yard"

	_, err := newListResolver(service).ServiceFailures(
		listContext(t),
		gqlmodel.DataTableConnectionInput{},
		nil,
		&raw,
	)
	require.Error(t, err)
	assert.Empty(t, service.requests)
}
