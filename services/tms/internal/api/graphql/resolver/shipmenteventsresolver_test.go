package resolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryResolver_ShipmentEvents_MapsRequest(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	shipmentID := pulid.MustNew("shp_")
	eventID := pulid.MustNew("se_")
	service := &recordingShipmentEventService{
		events: []*shipmentevent.Event{
			{
				ID:             eventID,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				ShipmentID:     shipmentID,
				Type:           shipmentevent.TypeStatusChanged,
				Severity:       shipmentevent.SeverityBrand,
				ActorType:      shipmentevent.ActorSystem,
				Summary:        "Status changed",
				OccurredAt:     1_800_000_000,
			},
		},
	}
	permissionEngine := &recordingPermissionEngine{}
	resolver := &queryResolver{&Resolver{
		shipmentEventService: service,
		permissionEngine:     permissionEngine,
	}}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})
	limit := 20
	before := 1_700_000_000
	types := []gqlmodel.ShipmentEventType{
		gqlmodel.ShipmentEventTypeStatusChanged,
		gqlmodel.ShipmentEventTypeCommentPosted,
	}

	events, err := resolver.ShipmentEvents(ctx, new(shipmentID.String()), types, &limit, &before)
	require.NoError(t, err)

	require.Len(t, events, 1)
	assert.Equal(t, eventID.String(), events[0].ID)
	require.NotNil(t, permissionEngine.request)
	assert.Equal(t, permission.ResourceShipment.String(), permissionEngine.request.Resource)
	assert.Equal(t, permission.OpRead, permissionEngine.request.Operation)
	require.NotNil(t, service.req)
	assert.Equal(t, orgID, service.req.TenantInfo.OrgID)
	assert.Equal(t, buID, service.req.TenantInfo.BuID)
	assert.Equal(t, userID, service.req.TenantInfo.UserID)
	assert.Equal(t, shipmentID, service.req.ShipmentID)
	assert.Equal(t, []shipmentevent.Type{
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeCommentPosted,
	}, service.req.Types)
	assert.Equal(t, 20, service.req.Limit)
	assert.Equal(t, int64(1_700_000_000), service.req.Before)
}

func TestQueryResolver_ShipmentEvents_AllowsOptionalFilters(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	service := &recordingShipmentEventService{}
	resolver := &queryResolver{&Resolver{
		shipmentEventService: service,
		permissionEngine:     &recordingPermissionEngine{},
	}}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	events, err := resolver.ShipmentEvents(ctx, nil, nil, nil, nil)
	require.NoError(t, err)

	assert.Empty(t, events)
	require.NotNil(t, service.req)
	assert.True(t, service.req.ShipmentID.IsNil())
	assert.Empty(t, service.req.Types)
	assert.Zero(t, service.req.Limit)
	assert.Zero(t, service.req.Before)
}

type recordingShipmentEventService struct {
	req    *repositories.ListShipmentEventsRequest
	events []*shipmentevent.Event
}

func (s *recordingShipmentEventService) Record(
	context.Context,
	*servicesport.RecordShipmentEventParams,
) error {
	return nil
}

func (s *recordingShipmentEventService) List(
	_ context.Context,
	req *repositories.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	s.req = req
	return s.events, nil
}

type recordingPermissionEngine struct {
	mocks.AllowAllPermissionEngine
	request *servicesport.PermissionCheckRequest
}

func (e *recordingPermissionEngine) Check(
	_ context.Context,
	req *servicesport.PermissionCheckRequest,
) (*servicesport.PermissionCheckResult, error) {
	e.request = req
	return &servicesport.PermissionCheckResult{Allowed: true}, nil
}
