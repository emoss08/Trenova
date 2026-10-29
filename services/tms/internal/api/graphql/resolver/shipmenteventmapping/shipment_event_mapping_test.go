package shipmenteventmapping

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentEventToModel_MapsOptionalFieldsAndRelations(t *testing.T) {
	t.Parallel()

	eventID := pulid.MustNew("se_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")
	stopID := pulid.MustNew("stp_")
	actorID := pulid.MustNew("usr_")
	actor := &tenant.User{
		ID:           actorID,
		Name:         "Maria Diaz",
		EmailAddress: "maria@example.com",
	}
	event := &shipmentevent.Event{
		ID:             eventID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     shipmentID,
		StopID:         stopID,
		Type:           shipmentevent.TypeStopCompleted,
		Severity:       shipmentevent.SeveritySuccess,
		ActorType:      shipmentevent.ActorUser,
		ActorID:        actorID,
		ActorLabel:     "Maria Diaz",
		Summary:        "Stop completed",
		Metadata:       map[string]any{"stopSequence": float64(2)},
		OccurredAt:     1_800_000_000,
		CorrelationID:  "corr-1",
		Actor:          actor,
		Shipment: &shipmentdomain.Shipment{
			ID:        shipmentID,
			ProNumber: "SHP-100",
		},
	}

	result, err := shipmentEventToModel(event)
	require.NoError(t, err)
	model, ok := result.(*gqlmodel.ShipmentMoveEvent)
	require.True(t, ok, "StopCompleted must map to ShipmentMoveEvent, got %T", result)

	assert.Equal(t, eventID.String(), model.ID)
	assert.Equal(t, orgID.String(), model.OrganizationID)
	assert.Equal(t, buID.String(), model.BusinessUnitID)
	assert.Equal(t, shipmentID.String(), model.ShipmentID)
	assert.Nil(t, model.MoveID)
	assert.Equal(t, stopID.String(), *model.StopID)
	assert.Nil(t, model.PreviousStatus)
	assert.Nil(t, model.NewStatus)
	assert.Equal(t, gqlmodel.ShipmentEventTypeStopCompleted, model.Type)
	assert.Equal(t, gqlmodel.ShipmentEventSeveritySuccess, model.Severity)
	assert.Equal(t, gqlmodel.ShipmentEventActorTypeUser, model.ActorType)
	assert.Equal(t, actorID.String(), *model.ActorID)
	assert.Equal(t, "Maria Diaz", model.ActorLabel)
	assert.Equal(t, "Stop completed", model.Summary)
	assert.Equal(t, map[string]any{"stopSequence": float64(2)}, model.Metadata)
	assert.Equal(t, 1_800_000_000, model.OccurredAt)
	assert.Equal(t, "corr-1", *model.CorrelationID)
	assert.Same(t, actor, model.Actor)
	require.NotNil(t, model.Shipment)
	assert.Equal(t, shipmentID.String(), *model.Shipment.ID)
	assert.Equal(t, "SHP-100", *model.Shipment.ProNumber)
}

func TestShipmentEventToModel_LiftsCommentPayload(t *testing.T) {
	t.Parallel()

	commentID := pulid.MustNew("scm_")
	event := &shipmentevent.Event{
		ID:        pulid.MustNew("se_"),
		CommentID: commentID,
		Type:      shipmentevent.TypeCommentPosted,
		Severity:  shipmentevent.SeverityInfo,
		ActorType: shipmentevent.ActorUser,
		Summary:   "Comment posted",
		Metadata: map[string]any{
			"commentBody":       "hello @ops",
			"commentType":       "Dispatch",
			"commentVisibility": "Operations",
			"commentPriority":   "High",
			"mentionedUserIds":  []string{"usr_1", "usr_2"},
		},
		OccurredAt: 1_800_000_000,
	}

	result, err := shipmentEventToModel(event)
	require.NoError(t, err)
	model, ok := result.(*gqlmodel.ShipmentCommentEvent)
	require.True(t, ok, "CommentPosted must map to ShipmentCommentEvent, got %T", result)

	assert.Equal(t, commentID.String(), *model.CommentID)
	assert.Equal(t, "hello @ops", *model.CommentBody)
	assert.Equal(t, gqlmodel.ShipmentCommentTypeDispatch, *model.CommentType)
	assert.Equal(t, gqlmodel.ShipmentCommentVisibilityOperations, *model.CommentVisibility)
	assert.Equal(t, gqlmodel.ShipmentCommentPriorityHigh, *model.CommentPriority)
	assert.Equal(t, []string{"usr_1", "usr_2"}, model.MentionedUserIds)
	assert.Equal(t, "hello @ops", model.Metadata["commentBody"])
}

func TestShipmentEventToModel_LiftsTenderPayloadFromMetadata(t *testing.T) {
	t.Parallel()

	event := &shipmentevent.Event{
		ID:        pulid.MustNew("se_"),
		Type:      shipmentevent.TypeTenderEntrySkipped,
		Severity:  shipmentevent.SeverityDanger,
		ActorType: shipmentevent.ActorSystem,
		Summary:   "Skipped Sunset Logistics",
		Metadata: map[string]any{
			"tenderId":    "tdr_1",
			"moveId":      "smv_1",
			"offerId":     "tof_1",
			"carrierName": "Sunset Logistics",
			"rank":        int16(3),
			"reasons":     []string{"Insurance policy expired", ""},
		},
		OccurredAt: 1_800_000_000,
	}

	result, err := shipmentEventToModel(event)
	require.NoError(t, err)
	model, ok := result.(*gqlmodel.ShipmentTenderEvent)
	require.True(t, ok, "TenderEntrySkipped must map to ShipmentTenderEvent, got %T", result)

	assert.Equal(t, "tdr_1", *model.TenderID)
	assert.Equal(t, "smv_1", *model.MoveID)
	assert.Equal(t, "tof_1", *model.OfferID)
	assert.Equal(t, "Sunset Logistics", *model.CarrierName)
	assert.Equal(t, 3, *model.Rank)
	assert.Equal(t, []string{"Insurance policy expired"}, model.Reasons)
	assert.Empty(t, model.Warnings)
	assert.Nil(t, model.Reason)
	assert.Nil(t, model.Channel)
}

func TestShipmentEventToModel_TenderMoveFallsBackToColumn(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	event := &shipmentevent.Event{
		ID:         pulid.MustNew("se_"),
		MoveID:     moveID,
		Type:       shipmentevent.TypeTenderWithdrawn,
		Severity:   shipmentevent.SeverityMuted,
		ActorType:  shipmentevent.ActorUser,
		Summary:    "Tender withdrawn",
		Metadata:   map[string]any{"tenderId": "tdr_1", "reason": "Covered elsewhere"},
		OccurredAt: 1_800_000_000,
	}

	result, err := shipmentEventToModel(event)
	require.NoError(t, err)
	model, ok := result.(*gqlmodel.ShipmentTenderEvent)
	require.True(t, ok)

	assert.Equal(t, moveID.String(), *model.MoveID)
	assert.Equal(t, "Covered elsewhere", *model.Reason)
	assert.Nil(t, model.Rank)
}

func TestShipmentEventToModel_LiftsHoldAndCarrierPayloads(t *testing.T) {
	t.Parallel()

	holdID := pulid.MustNew("hld_")
	holdEvent := &shipmentevent.Event{
		ID:        pulid.MustNew("se_"),
		HoldID:    holdID,
		Type:      shipmentevent.TypeHoldPlaced,
		Severity:  shipmentevent.SeverityDanger,
		ActorType: shipmentevent.ActorUser,
		Summary:   "Hold placed",
		Metadata: map[string]any{
			"holdType":     "ComplianceHold",
			"holdSeverity": "Blocking",
			"holdSource":   "Rule",
		},
		OccurredAt: 1_800_000_000,
	}

	holdResult, err := shipmentEventToModel(holdEvent)
	require.NoError(t, err)
	hold, ok := holdResult.(*gqlmodel.ShipmentHoldEvent)
	require.True(t, ok, "HoldPlaced must map to ShipmentHoldEvent, got %T", holdResult)
	assert.Equal(t, holdID.String(), *hold.HoldID)
	assert.Equal(t, holdreason.HoldTypeCompliance, *hold.HoldType)
	assert.Equal(t, holdreason.HoldSeverityBlocking, *hold.HoldSeverity)
	assert.Equal(t, "Rule", *hold.HoldSource)

	moveID := pulid.MustNew("smv_")
	carrierEvent := &shipmentevent.Event{
		ID:        pulid.MustNew("se_"),
		MoveID:    moveID,
		Type:      shipmentevent.TypeCarrierAssigned,
		Severity:  shipmentevent.SeverityMuted,
		ActorType: shipmentevent.ActorUser,
		Summary:   "Carrier assigned",
		Metadata: map[string]any{
			"carrierName": "Blue Ridge Freight",
			"carrierId":   "car_1",
			"totalCost":   "2450.00",
			"proNumber":   "BRF-88213",
		},
		OccurredAt: 1_800_000_000,
	}

	carrierResult, err := shipmentEventToModel(carrierEvent)
	require.NoError(t, err)
	carrier, ok := carrierResult.(*gqlmodel.ShipmentCarrierEvent)
	require.True(t, ok, "CarrierAssigned must map to ShipmentCarrierEvent, got %T", carrierResult)
	assert.Equal(t, moveID.String(), *carrier.MoveID)
	assert.Equal(t, "car_1", *carrier.CarrierID)
	assert.Equal(t, "Blue Ridge Freight", *carrier.CarrierName)
	assert.Equal(t, "2450.00", *carrier.TotalCost)
	assert.Equal(t, "BRF-88213", *carrier.ProNumber)
	assert.Nil(t, carrier.Reason)
}

func TestShipmentEventToModel_RejectsUnknownType(t *testing.T) {
	t.Parallel()

	_, err := shipmentEventToModel(&shipmentevent.Event{
		ID:   pulid.MustNew("se_"),
		Type: shipmentevent.Type("NotARealShipmentEventType"),
	})
	require.Error(t, err)
}
