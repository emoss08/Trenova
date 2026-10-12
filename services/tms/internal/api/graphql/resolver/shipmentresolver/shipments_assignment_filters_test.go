package shipmentresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listCapturingShipmentService struct {
	shipmentports.ShipmentService

	requests []*shipmentports.ListShipmentsRequest
}

func (s *listCapturingShipmentService) List(
	_ context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipmentdomain.Shipment], error) {
	s.requests = append(s.requests, req)
	return &pagination.CursorListResult[*shipmentdomain.Shipment]{}, nil
}

func shipmentListContext(t *testing.T) context.Context {
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

func newShipmentListResolver(service shipmentports.ShipmentService) *QueryResolver {
	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		permission.ResourceShipment.String() + "|" + string(permission.OpRead): true,
	}}
	return &QueryResolver{&Deps{
		Core:            &base.Core{PermissionEngine: engine},
		ShipmentService: service,
	}}
}

func TestShipments_TractorIDsReachTheListOptions(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}
	tractorA := pulid.MustNew("tr_")
	tractorB := pulid.MustNew("tr_")
	first := 5

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{
			First:      &first,
			TractorIds: []string{tractorA.String(), tractorB.String()},
		},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	assert.Equal(
		t,
		[]pulid.ID{tractorA, tractorB},
		service.requests[0].ShipmentOptions.TractorIDs,
	)
}

func TestShipments_TrailerAndWorkerIDsReachTheListOptions(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}
	trailerID := pulid.MustNew("trl_")
	workerID := pulid.MustNew("wrk_")

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{
			TrailerIds: []string{trailerID.String()},
			WorkerIds:  []string{workerID.String()},
		},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	options := service.requests[0].ShipmentOptions
	assert.Equal(t, []pulid.ID{trailerID}, options.TrailerIDs)
	assert.Equal(t, []pulid.ID{workerID}, options.WorkerIDs)
	assert.Empty(t, options.TractorIDs)
}

func TestShipments_NoTractorIDsLeavesTheFilterOff(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	assert.Empty(t, service.requests[0].ShipmentOptions.TractorIDs)
}

func TestShipments_RejectsAMalformedTractorID(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{TractorIds: []string{"tractor-one"}},
	)
	require.Error(t, err)
	assert.Empty(t, service.requests)
}
