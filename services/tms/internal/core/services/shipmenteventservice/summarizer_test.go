package shipmenteventservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTenant(t *testing.T) TenantRef {
	t.Helper()
	return TenantRef{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func userActor() services.AuditActor {
	userID := pulid.MustNew("usr_")
	return services.AuditActor{
		PrincipalType: services.PrincipalTypeUser,
		PrincipalID:   userID,
		UserID:        userID,
	}
}

func TestBuildShipmentCreated(t *testing.T) {
	t.Parallel()

	tenant := newTenant(t)
	sh := &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "PRO-1042",
		Status:    shipment.StatusNew,
	}

	params := BuildShipmentCreated(tenant, sh, userActor())

	require.NotNil(t, params)
	assert.Equal(t, shipmentevent.TypeShipmentCreated, params.Type)
	assert.Equal(t, shipmentevent.SeverityMuted, params.Severity)
	assert.Equal(t, sh.ID, params.ShipmentID)
	assert.Equal(t, "Shipment created", params.Summary)
	assert.Equal(t, "PRO-1042", params.Metadata["proNumber"])
}

func TestBuildStatusChangedSeverity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		status   shipment.Status
		expected shipmentevent.Severity
	}{
		{"InTransit -> brand", shipment.StatusInTransit, shipmentevent.SeverityBrand},
		{"Completed -> success", shipment.StatusCompleted, shipmentevent.SeveritySuccess},
		{"Canceled -> danger", shipment.StatusCanceled, shipmentevent.SeverityDanger},
		{"Delayed -> danger", shipment.StatusDelayed, shipmentevent.SeverityDanger},
		{"New -> muted", shipment.StatusNew, shipmentevent.SeverityMuted},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tenant := newTenant(t)
			sh := &shipment.Shipment{
				ID:        pulid.MustNew("shp_"),
				ProNumber: "PRO-1",
				Status:    tc.status,
			}
			params := BuildStatusChanged(tenant, sh, shipment.StatusNew, userActor())
			assert.Equal(t, shipmentevent.TypeStatusChanged, params.Type)
			assert.Equal(t, tc.expected, params.Severity)
		})
	}
}

func TestBuildShipmentCanceledCarriesReason(t *testing.T) {
	t.Parallel()

	sh := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: "PRO-1"}
	tenant := newTenant(t)

	params := BuildShipmentCanceled(tenant, sh, "Customer request", userActor())
	assert.Equal(t, shipmentevent.TypeShipmentCanceled, params.Type)
	assert.Equal(t, shipmentevent.SeverityDanger, params.Severity)
	assert.Equal(t, "Shipment canceled", params.Summary)
	assert.Equal(t, "Customer request", params.Metadata["reason"])
	assert.Equal(t, "PRO-1", params.Metadata["proNumber"])
}

func TestBuildMoveStatusChangedClassification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		newStatus   shipment.MoveStatus
		expectType  shipmentevent.Type
		expectSever shipmentevent.Severity
	}{
		{
			"InTransit -> Departed",
			shipment.MoveStatusInTransit,
			shipmentevent.TypeMoveDeparted,
			shipmentevent.SeverityBrand,
		},
		{
			"Completed -> Arrived",
			shipment.MoveStatusCompleted,
			shipmentevent.TypeMoveArrived,
			shipmentevent.SeveritySuccess,
		},
		{
			"Canceled -> Danger",
			shipment.MoveStatusCanceled,
			shipmentevent.TypeMoveStatusChanged,
			shipmentevent.SeverityDanger,
		},
		{
			"Assigned -> Generic",
			shipment.MoveStatusAssigned,
			shipmentevent.TypeMoveStatusChanged,
			shipmentevent.SeverityMuted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			move := &shipment.ShipmentMove{
				ID:         pulid.MustNew("sm_"),
				ShipmentID: pulid.MustNew("shp_"),
				Status:     tc.newStatus,
			}
			params := BuildMoveStatusChanged(
				newTenant(t),
				move,
				shipment.MoveStatusNew,
				userActor(),
			)

			assert.Equal(t, tc.expectType, params.Type)
			assert.Equal(t, tc.expectSever, params.Severity)
			assert.Equal(t, move.ID, params.MoveID)
			assert.Equal(t, move.ShipmentID, params.ShipmentID)
		})
	}
}

func TestBuildHoldEvents(t *testing.T) {
	t.Parallel()

	hold := &shipment.ShipmentHold{
		ID:         pulid.MustNew("shh_"),
		ShipmentID: pulid.MustNew("shp_"),
		Type:       "Operational",
		Source:     shipment.HoldSourceUser,
	}
	tenant := newTenant(t)

	placed := BuildHoldPlaced(tenant, hold, userActor())
	assert.Equal(t, shipmentevent.SeverityDanger, placed.Severity)
	assert.Equal(t, shipmentevent.TypeHoldPlaced, placed.Type)

	released := BuildHoldReleased(tenant, hold, userActor())
	assert.Equal(t, shipmentevent.SeveritySuccess, released.Severity)
	assert.Equal(t, shipmentevent.TypeHoldReleased, released.Type)

	updated := BuildHoldUpdated(tenant, hold, userActor())
	assert.Equal(t, shipmentevent.SeverityMuted, updated.Severity)
	assert.Equal(t, shipmentevent.TypeHoldUpdated, updated.Type)
}

func TestBuildCommentPostedCarriesBody(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("a", 250)
	comment := &shipment.ShipmentComment{
		ID:         pulid.MustNew("shc_"),
		ShipmentID: pulid.MustNew("shp_"),
		Comment:    body,
		Type:       shipment.CommentTypeInternal,
	}

	params := BuildCommentPosted(newTenant(t), comment, userActor())

	assert.Equal(t, shipmentevent.TypeCommentPosted, params.Type)
	assert.Equal(t, shipmentevent.SeverityInfo, params.Severity)
	assert.Equal(t, "Comment added", params.Summary)
	assert.Equal(t, body, params.Metadata["commentBody"])
	assert.Equal(t, comment.ID, params.CommentID)
}

func TestBuildDriverEvents(t *testing.T) {
	t.Parallel()

	tenant := newTenant(t)
	workerID := pulid.MustNew("wkr_")
	tractorID := pulid.MustNew("trt_")
	assignment := &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		ShipmentMoveID:  pulid.MustNew("sm_"),
		PrimaryWorkerID: &workerID,
		TractorID:       &tractorID,
	}
	ref := AssignmentRef{
		ShipmentID:   pulid.MustNew("shp_"),
		MoveID:       assignment.ShipmentMoveID,
		AssignmentID: assignment.ID,
	}

	assigned := BuildDriverAssigned(tenant, ref, assignment, "S. Ndiaye", userActor())
	assert.Equal(t, shipmentevent.TypeDriverAssigned, assigned.Type)
	assert.Equal(t, "Driver assigned", assigned.Summary)
	assert.Equal(t, "S. Ndiaye", assigned.Metadata["driverName"])
	assert.Equal(t, ref.ShipmentID, assigned.ShipmentID)
	assert.Equal(t, ref.AssignmentID, assigned.AssignmentID)

	reassigned := BuildDriverReassigned(tenant, ref, assignment, "S. Ndiaye", userActor())
	assert.Equal(t, shipmentevent.TypeDriverReassigned, reassigned.Type)
	assert.Equal(t, "Driver reassigned", reassigned.Summary)
	assert.Equal(t, "S. Ndiaye", reassigned.Metadata["driverName"])

	unassigned := BuildDriverUnassigned(tenant, ref, userActor())
	assert.Equal(t, shipmentevent.TypeDriverUnassigned, unassigned.Type)
	assert.Equal(t, ref.AssignmentID, unassigned.AssignmentID)
}

func TestResolveActor(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	at, id := resolveActor(services.AuditActor{
		PrincipalType: services.PrincipalTypeUser,
		PrincipalID:   userID,
		UserID:        userID,
	})
	assert.Equal(t, shipmentevent.ActorUser, at)
	assert.Equal(t, userID, id)

	keyID := pulid.MustNew("ak_")
	at, id = resolveActor(services.AuditActor{
		PrincipalType: services.PrincipalTypeAPIKey,
		PrincipalID:   keyID,
		APIKeyID:      keyID,
	})
	assert.Equal(t, shipmentevent.ActorAPIKey, at)
	assert.Equal(t, keyID, id)

	at, id = resolveActor(services.AuditActor{})
	assert.Equal(t, shipmentevent.ActorSystem, at)
	assert.True(t, id.IsNil())
}
