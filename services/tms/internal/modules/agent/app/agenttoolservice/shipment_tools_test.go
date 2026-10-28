package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCommentService struct {
	shipmentcommentports.ShipmentCommentService

	created *shipment.ShipmentComment
	err     error
}

func (f *fakeCommentService) Create(
	_ context.Context,
	entity *shipment.ShipmentComment,
	_ *permissionports.RequestActor,
) (*shipment.ShipmentComment, error) {
	f.created = entity

	return entity, f.err
}

type fakeHoldService struct {
	shipmentholdports.ShipmentHoldService

	placed   *shipmentholdports.CreateShipmentHoldRequest
	released *shipmentholdports.ReleaseShipmentHoldRequest
	err      error
}

func (f *fakeHoldService) Create(
	_ context.Context,
	req *shipmentholdports.CreateShipmentHoldRequest,
	_ *permissionports.RequestActor,
) (*shipment.ShipmentHold, error) {
	f.placed = req

	return &shipment.ShipmentHold{ID: pulid.MustNew("shld_")}, f.err
}

func (f *fakeHoldService) Release(
	_ context.Context,
	req *shipmentholdports.ReleaseShipmentHoldRequest,
	_ *permissionports.RequestActor,
) (*shipment.ShipmentHold, error) {
	f.released = req

	return &shipment.ShipmentHold{ID: req.HoldID}, f.err
}

type fakeShipmentService struct {
	shipmentserviceports.ShipmentService

	canceled *shipmentports.CancelShipmentRequest
	err      error
}

func (f *fakeShipmentService) Cancel(
	_ context.Context,
	req *shipmentports.CancelShipmentRequest,
	_ *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	f.canceled = req

	return &shipment.Shipment{ID: req.ShipmentID}, f.err
}

/*
A comment the assistant leaves has to be marked as the assistant's.

The shipment comment thread is read by dispatchers deciding what to do next. A
note that looks hand-typed but was not is worse than no note: it carries the
authority of a colleague without anyone having checked it. CommentSourceAI is
already in the domain for exactly this.
*/
func TestAddShipmentComment_MarksItselfAsAIWritten(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentService{}
	tool := shipmenttools.NewAddShipmentCommentTool(comments)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"comment":    "Carrier confirmed a 14:00 pickup.",
	}))
	require.NoError(t, err)
	require.NotNil(t, comments.created)

	assert.Equal(t, shipment.CommentSourceAI, comments.created.Source)
	assert.Equal(t, "Carrier confirmed a 14:00 pickup.", comments.created.Comment)
}

// Internal is the only safe default: a comment the agent writes must not reach
// a customer or a driver unless someone chose that deliberately.
func TestAddShipmentComment_DefaultsToInternalVisibility(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentService{}
	tool := shipmenttools.NewAddShipmentCommentTool(comments)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"comment":    "Checked the BOL.",
	})))

	assert.Equal(t, shipment.CommentVisibilityInternal, comments.created.Visibility)
}

func TestAddShipmentComment_CarriesTheAuthorAndTenant(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentService{}
	tool := shipmenttools.NewAddShipmentCommentTool(comments)

	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"comment":    "Noted.",
	})
	require.NoError(t, tool.Execute(t.Context(), params))

	assert.Equal(t, params.OrganizationID, comments.created.OrganizationID)
	assert.Equal(t, params.BusinessUnitID, comments.created.BusinessUnitID)
	assert.Equal(t, params.Actor.UserID, comments.created.UserID)
}

func TestAddShipmentComment_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentService{}
	tool := shipmenttools.NewAddShipmentCommentTool(comments)

	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"comment":    "Noted.",
	})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrTenantMismatch)
	assert.Nil(t, comments.created)
}

/*
A hold stops a shipment moving, billing or delivering. The reason is not
decoration: it is what the hold's blocking behaviour is derived from, and it is
what the person releasing it reads. Requiring a real reason id keeps the agent
from inventing a category the organization does not use.
*/
func TestPlaceShipmentHold_RequiresARealReason(t *testing.T) {
	t.Parallel()

	holds := &fakeHoldService{}
	tool := shipmenttools.NewPlaceShipmentHoldTool(holds)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"note":       "Awaiting a corrected BOL.",
	}))
	require.Error(t, err)
	assert.Nil(t, holds.placed)
}

func TestPlaceShipmentHold_PassesTheReasonAndNotes(t *testing.T) {
	t.Parallel()

	holds := &fakeHoldService{}
	tool := shipmenttools.NewPlaceShipmentHoldTool(holds)

	shipmentID := pulid.MustNew("shp_")
	reasonID := pulid.MustNew("hrsn_")

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId":   shipmentID.String(),
		"holdReasonId": reasonID.String(),
		"note":         "Awaiting a corrected BOL.",
	})))

	require.NotNil(t, holds.placed)
	assert.Equal(t, shipmentID, holds.placed.ShipmentID)
	assert.Equal(t, reasonID, holds.placed.HoldReasonID)
	assert.Equal(t, "Awaiting a corrected BOL.", holds.placed.Notes)
}

// The blocking flags are left unset so the hold reason's own defaults apply.
// An agent choosing them would be deciding policy the organization already
// configured.
func TestPlaceShipmentHold_LeavesBlockingToTheReason(t *testing.T) {
	t.Parallel()

	holds := &fakeHoldService{}
	tool := shipmenttools.NewPlaceShipmentHoldTool(holds)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId":   pulid.MustNew("shp_").String(),
		"holdReasonId": pulid.MustNew("hrsn_").String(),
	})))

	assert.Nil(t, holds.placed.BlocksDispatch)
	assert.Nil(t, holds.placed.BlocksBilling)
	assert.Nil(t, holds.placed.VisibleToCustomer)
}

func TestReleaseShipmentHold_PassesBothIdentifiers(t *testing.T) {
	t.Parallel()

	holds := &fakeHoldService{}
	tool := shipmenttools.NewReleaseShipmentHoldTool(holds)

	shipmentID := pulid.MustNew("shp_")
	holdID := pulid.MustNew("shld_")

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": shipmentID.String(),
		"holdId":     holdID.String(),
	})))

	require.NotNil(t, holds.released)
	assert.Equal(t, shipmentID, holds.released.ShipmentID)
	assert.Equal(t, holdID, holds.released.HoldID)
}

/*
Cancelling is the most consequential thing in this batch and the least
reversible: the customer is told, the drivers are released, and the revenue is
gone. It requires a reason, proposes rather than acting, and reports itself as
irreversible so the proposal card says so.
*/
func TestCancelShipment_RequiresAReason(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentService{}
	tool := shipmenttools.NewCancelShipmentTool(shipments, nil, nil)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
	}))
	require.Error(t, err)
	assert.Nil(t, shipments.canceled, "a cancellation without a reason never reaches the service")
}

func TestCancelShipment_RecordsWhoCanceledIt(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentService{}
	tool := shipmenttools.NewCancelShipmentTool(shipments, nil, nil)

	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"reason":     "Customer cancelled the order.",
	})
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, shipments.canceled)
	assert.Equal(t, params.Actor.UserID, shipments.canceled.CanceledByID)
	assert.Positive(t, shipments.canceled.CanceledAt)
	assert.Equal(t, "Customer cancelled the order.", shipments.canceled.CancelReason)
}

func TestCancelShipment_IsIrreversibleAndOnlyProposes(t *testing.T) {
	t.Parallel()

	tool := shipmenttools.NewCancelShipmentTool(&fakeShipmentService{}, nil, nil)

	assert.False(t, tool.Policy().Reversible)
	assert.Equal(t, agent.TierPropose, tool.Policy().DefaultTier)
	assert.Equal(t, permission.OpCancel, tool.Policy().Operation)
}

// A tool that surfaces a service failure as success would have the agent report
// a hold that was never placed.
func TestShipmentWriteTools_SurfaceServiceFailures(t *testing.T) {
	t.Parallel()

	failure := errors.New("shipment is already delivered")

	holds := &fakeHoldService{err: failure}
	require.ErrorIs(
		t,
		shipmenttools.NewPlaceShipmentHoldTool(holds).
			Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"shipmentId":   pulid.MustNew("shp_").String(),
				"holdReasonId": pulid.MustNew("hrsn_").String(),
			})),
		failure,
	)

	shipments := &fakeShipmentService{err: failure}
	require.ErrorIs(
		t,
		shipmenttools.NewCancelShipmentTool(shipments, nil, nil).
			Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"shipmentId": pulid.MustNew("shp_").String(),
				"reason":     "duplicate booking",
			})),
		failure,
	)
}

// Each write tool authorizes against the thing it writes, not the shipment it
// hangs off — otherwise granting an agent shipment:update would silently grant
// it the power to cancel.
func TestShipmentWriteTools_AuthorizeAgainstWhatTheyWrite(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tool      agenttoolports.AgentTool
		resource  permission.Resource
		operation permission.Operation
	}{
		{shipmenttools.NewAddShipmentCommentTool(nil), permission.ResourceShipmentComment, permission.OpCreate},
		{shipmenttools.NewPlaceShipmentHoldTool(nil), permission.ResourceShipmentHold, permission.OpCreate},
		{shipmenttools.NewReleaseShipmentHoldTool(nil), permission.ResourceShipmentHold, permission.OpUpdate},
		{shipmenttools.NewCancelShipmentTool(nil, nil, nil), permission.ResourceShipment, permission.OpCancel},
	} {
		assert.Equal(t, tc.resource, tc.tool.Policy().Resource, tc.tool.Name())
		assert.Equal(t, tc.operation, tc.tool.Policy().Operation, tc.tool.Name())
	}
}

/*
Type and visibility are separate axes, and leaving type pinned to Internal while
visibility widens produces a comment that contradicts itself: filed as an
internal note, shown to the customer. Whoever later filters the thread by type
would miss it.
*/
func TestAddShipmentComment_KeepsTypeAndVisibilityConsistent(t *testing.T) {
	t.Parallel()

	for visibility, expected := range map[shipment.CommentVisibility]shipment.CommentType{
		shipment.CommentVisibilityCustomer: shipment.CommentTypeCustomerUpdate,
		shipment.CommentVisibilityDriver:   shipment.CommentTypeDriverUpdate,
		shipment.CommentVisibilityInternal: shipment.CommentTypeInternal,
	} {
		comments := &fakeCommentService{}
		tool := shipmenttools.NewAddShipmentCommentTool(comments)

		require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
			"shipmentId": pulid.MustNew("shp_").String(),
			"comment":    "Update.",
			"visibility": string(visibility),
		})))

		assert.Equal(t, expected, comments.created.Type, "visibility %s", visibility)
		assert.Equal(t, visibility, comments.created.Visibility)
	}
}
