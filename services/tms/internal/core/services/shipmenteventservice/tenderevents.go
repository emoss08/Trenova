package shipmenteventservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// TenderRef wires a tender event to its shipment, move, and offer.
type TenderRef struct {
	ShipmentID pulid.ID
	MoveID     pulid.ID
	TenderID   pulid.ID
	OfferID    pulid.ID
}

func tenderMetadata(ref TenderRef, extra map[string]any) map[string]any {
	metadata := map[string]any{
		"tenderId": ref.TenderID.String(),
	}
	if !ref.MoveID.IsNil() {
		metadata["moveId"] = ref.MoveID.String()
	}
	if !ref.OfferID.IsNil() {
		metadata["offerId"] = ref.OfferID.String()
	}
	for k, v := range extra {
		metadata[k] = v
	}
	return metadata
}

func buildTenderEvent(
	tenant TenantRef,
	ref TenderRef,
	eventType shipmentevent.Type,
	severity shipmentevent.Severity,
	summary string,
	metadata map[string]any,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     ref.ShipmentID,
		Type:           eventType,
		Severity:       severity,
		Summary:        summary,
		Metadata:       tenderMetadata(ref, metadata),
		Actor:          actor,
	}
}

// BuildTenderOffered records one offer going out to a carrier.
func BuildTenderOffered(
	tenant TenantRef,
	ref TenderRef,
	carrierName string,
	rank int16,
	channel tender.Channel,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderOffered,
		shipmentevent.SeverityInfo,
		"Load offered to "+carrierName,
		map[string]any{
			"carrierName": carrierName,
			"rank":        rank,
			"channel":     string(channel),
		},
		actor,
	)
}

// BuildTenderAccepted records a carrier accepting an offer.
func BuildTenderAccepted(
	tenant TenantRef,
	ref TenderRef,
	carrierName string,
	source tender.ResponseSource,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderAccepted,
		shipmentevent.SeveritySuccess,
		carrierName+" accepted the tender",
		map[string]any{
			"carrierName": carrierName,
			"source":      string(source),
		},
		actor,
	)
}

// BuildTenderDeclined records a carrier declining an offer.
func BuildTenderDeclined(
	tenant TenantRef,
	ref TenderRef,
	carrierName, reason string,
	source tender.ResponseSource,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderDeclined,
		shipmentevent.SeverityInfo,
		carrierName+" declined the tender",
		map[string]any{
			"carrierName": carrierName,
			"reason":      reason,
			"source":      string(source),
		},
		actor,
	)
}

// BuildTenderExpired records an offer lapsing with no response.
func BuildTenderExpired(
	tenant TenantRef,
	ref TenderRef,
	carrierName string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderExpired,
		shipmentevent.SeverityMuted,
		"Offer to "+carrierName+" expired",
		map[string]any{"carrierName": carrierName},
		actor,
	)
}

// BuildTenderWithdrawn records a tender being canceled with offers
// outstanding.
func BuildTenderWithdrawn(
	tenant TenantRef,
	ref TenderRef,
	reason string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderWithdrawn,
		shipmentevent.SeverityMuted,
		"Tender withdrawn",
		map[string]any{"reason": reason},
		actor,
	)
}

// BuildTenderNeedsReview records an accepted offer that could not be
// auto-assigned.
func BuildTenderNeedsReview(
	tenant TenantRef,
	ref TenderRef,
	carrierName, reason string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderNeedsReview,
		shipmentevent.SeverityDanger,
		"Tender acceptance from "+carrierName+" needs review",
		map[string]any{
			"carrierName": carrierName,
			"reason":      reason,
		},
		actor,
	)
}

// BuildRoutingGuideExhausted records a tender running out of carriers.
func BuildRoutingGuideExhausted(
	tenant TenantRef,
	ref TenderRef,
	mode tender.Mode,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	summary := "Routing guide exhausted with no carrier accepting"
	if mode != tender.ModeWaterfall {
		summary = "Spot tender exhausted with no carrier accepting"
	}
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeRoutingGuideExhausted,
		shipmentevent.SeverityDanger,
		summary,
		map[string]any{"mode": string(mode)},
		actor,
	)
}

// BuildTenderLateResponse records a response that arrived after the offer had
// already reached a terminal state.
func BuildTenderLateResponse(
	tenant TenantRef,
	ref TenderRef,
	carrierName string,
	action tender.ResponseAction,
	source tender.ResponseSource,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderLateResponse,
		shipmentevent.SeverityMuted,
		"Late "+string(action)+" from "+carrierName+" recorded",
		map[string]any{
			"carrierName": carrierName,
			"action":      string(action),
			"source":      string(source),
		},
		actor,
	)
}

// BuildTenderEntrySkipped records a routing guide entry excluded from a new
// tender because its carrier failed the eligibility gate.
func BuildTenderEntrySkipped(
	tenant TenantRef,
	ref TenderRef,
	carrierName string,
	rank int16,
	reasons []string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderEntrySkipped,
		shipmentevent.SeverityDanger,
		carrierName+" was skipped on the routing guide: "+strings.Join(reasons, "; "),
		map[string]any{
			"carrierName": carrierName,
			"rank":        rank,
			"reasons":     reasons,
		},
		actor,
	)
}

// BuildTenderEntryWarned records a routing guide entry tendered despite
// insurance warnings on its carrier.
func BuildTenderEntryWarned(
	tenant TenantRef,
	ref TenderRef,
	carrierName string,
	rank int16,
	warnings []string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderEntryWarned,
		shipmentevent.SeverityInfo,
		carrierName+" was tendered with insurance warnings: "+strings.Join(warnings, "; "),
		map[string]any{
			"carrierName": carrierName,
			"rank":        rank,
			"warnings":    warnings,
		},
		actor,
	)
}

// BuildTenderDeliveryFailed records an offer that could not be dispatched on
// its channel.
func BuildTenderDeliveryFailed(
	tenant TenantRef,
	ref TenderRef,
	carrierName string,
	channel tender.Channel,
	deliveryError string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return buildTenderEvent(tenant, ref,
		shipmentevent.TypeTenderDeliveryFailed,
		shipmentevent.SeverityDanger,
		"Offer to "+carrierName+" could not be delivered",
		map[string]any{
			"carrierName": carrierName,
			"channel":     string(channel),
			"error":       deliveryError,
		},
		actor,
	)
}
