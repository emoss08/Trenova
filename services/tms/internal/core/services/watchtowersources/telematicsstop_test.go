package watchtowersources_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stopReview(moveID, stopID pulid.ID, occurredAt int64, reason string) *repositories.StopReview {
	return &repositories.StopReview{
		ShipmentID: pulid.MustNew("shp_"),
		Event: &telematics.TelematicsEvent{
			ID:                telematics.NewEventID(),
			OrganizationID:    pulid.MustNew("org_"),
			BusinessUnitID:    pulid.MustNew("bu_"),
			OccurredAt:        occurredAt,
			AddressName:       "Acme DC",
			StopOutcome:       telematics.StopOutcomeRefused,
			StopVisit:         shipment.VisitDeparture,
			ShipmentMoveID:    moveID,
			StopID:            stopID,
			StopOutcomeReason: reason,
		},
	}
}

func TestDescribeTelematicsStopVisits_OneItemPerStopVisit(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	latest := stopReview(moveID, stopID, 300, "latest refusal")
	earlier := stopReview(moveID, stopID, 200, "earlier refusal")
	other := stopReview(moveID, pulid.MustNew("stp_"), 100, "other stop")

	items := watchtowersources.DescribeTelematicsStopVisits(
		[]*repositories.StopReview{latest, earlier, nil, other},
	)

	require.Len(t, items, 2)
	assert.Equal(t, "latest refusal", items[0].Summary)
	assert.Equal(t, moveID.String()+":"+stopID.String()+":Departure", items[0].SourceID)
	assert.Equal(t, watchtower.SourceTelematicsStopVisit, items[0].SourceKind)
	assert.Equal(t, watchtower.SeverityWarning, items[0].Severity)
	assert.Equal(t, "Departure at Acme DC not recorded", items[0].Title)
	assert.True(t, stringutils.IsSafeAppPath(items[0].Path))
	assert.Contains(t, items[0].Path, latest.ShipmentID.String())
	assert.Equal(t, "other stop", items[1].Summary)
}

func TestTelematicsStopVisitSourceID_FallsBackToTheLocation(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	locationID := pulid.MustNew("loc_")
	event := &telematics.TelematicsEvent{
		ShipmentMoveID: moveID,
		LocationID:     locationID,
		StopVisit:      shipment.VisitArrival,
	}
	assert.Equal(t, moveID.String()+":"+locationID.String()+":Arrival",
		watchtowersources.TelematicsStopVisitSourceID(event))

	event.LocationID = pulid.Nil
	assert.Equal(t, moveID.String()+":unlocated:Arrival",
		watchtowersources.TelematicsStopVisitSourceID(event))
}
