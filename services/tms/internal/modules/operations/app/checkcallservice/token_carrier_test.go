package checkcallservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const carrierBoundToken = "carrier-bound-token"

func seedLink(h *harness) {
	h.links.links = append(h.links.links, &checkcall.TrackingLink{
		ID:             checkcall.NewTrackingLinkID(),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		ShipmentID:     h.move.ShipmentID,
		ShipmentMoveID: h.move.ID,
		CarrierID:      h.carrierID,
		TokenHash:      tokenutils.Hash(carrierBoundToken),
		ExpiresAt:      testNow + 3600,
	})
}

func TestPreviewByTokenAFinishedLoadStillChecksTheCarrier(t *testing.T) {
	t.Parallel()

	for name, status := range map[string]shipmentvocab.MoveStatus{
		"completed": shipmentvocab.MoveStatusCompleted,
		"canceled":  shipmentvocab.MoveStatusCanceled,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			seedLink(h)
			h.move.Status = status
			h.assignments.assignment = &shipment.CarrierAssignment{
				CarrierID: pulid.MustNew("car_"),
				Status:    shipmentvocab.CarrierAssignmentStatusConfirmed,
			}

			_, err := h.svc.PreviewByToken(t.Context(), carrierBoundToken)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "This tracking link is no longer valid")
		})
	}
}

func TestPreviewByTokenAFinishedLoadStaysReadableForItsCarrier(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	seedLink(h)
	h.move.Status = shipmentvocab.MoveStatusCompleted

	view, err := h.svc.PreviewByToken(t.Context(), carrierBoundToken)

	require.NoError(t, err)
	assert.True(t, view.Closed)
}

func TestPreviewByTokenAFinishedLoadWithNoAssignmentStaysReadable(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	seedLink(h)
	h.move.Status = shipmentvocab.MoveStatusCompleted
	h.assignments.assignment = nil

	view, err := h.svc.PreviewByToken(t.Context(), carrierBoundToken)

	require.NoError(t, err)
	assert.True(t, view.Closed)
}

func TestPreviewByTokenAnOpenLoadWithNoAssignmentIsADeadLink(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	seedLink(h)
	h.assignments.assignment = nil

	_, err := h.svc.PreviewByToken(t.Context(), carrierBoundToken)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "This tracking link is no longer valid")
}
