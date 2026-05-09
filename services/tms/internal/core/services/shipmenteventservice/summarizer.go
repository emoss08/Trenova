package shipmenteventservice

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// TenantRef bundles the tenant identifiers required for every event.
type TenantRef struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

// AssignmentRef captures the IDs needed to wire an assignment event to its
// shipment, since the Assignment domain entity only links to the move.
type AssignmentRef struct {
	ShipmentID   pulid.ID
	MoveID       pulid.ID
	AssignmentID pulid.ID
}

// BuildShipmentCreated builds an event for shipment creation.
func BuildShipmentCreated(
	tenant TenantRef,
	sh *shipment.Shipment,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     sh.ID,
		Type:           shipmentevent.TypeShipmentCreated,
		Severity:       shipmentevent.SeverityMuted,
		Summary:        "Shipment created",
		Metadata: map[string]any{
			"proNumber": sh.ProNumber,
			"status":    string(sh.Status),
		},
		Actor: actor,
	}
}

// BuildStatusChanged emits a status transition event.
// Selects severity based on the new status.
func BuildStatusChanged(
	tenant TenantRef,
	sh *shipment.Shipment,
	previous shipment.Status,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     sh.ID,
		Type:           shipmentevent.TypeStatusChanged,
		Severity:       severityForShipmentStatus(sh.Status),
		Summary:        "Status updated",
		Metadata: map[string]any{
			"proNumber":      sh.ProNumber,
			"previousStatus": string(previous),
			"newStatus":      string(sh.Status),
		},
		Actor: actor,
	}
}

// BuildShipmentCanceled emits a cancellation event.
func BuildShipmentCanceled(
	tenant TenantRef,
	sh *shipment.Shipment,
	reason string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     sh.ID,
		Type:           shipmentevent.TypeShipmentCanceled,
		Severity:       shipmentevent.SeverityDanger,
		Summary:        "Shipment canceled",
		Metadata: map[string]any{
			"proNumber": sh.ProNumber,
			"reason":    reason,
		},
		Actor: actor,
	}
}

// BuildShipmentUncanceled emits an uncancel event.
func BuildShipmentUncanceled(
	tenant TenantRef,
	sh *shipment.Shipment,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     sh.ID,
		Type:           shipmentevent.TypeShipmentUncanceled,
		Severity:       shipmentevent.SeverityBrand,
		Summary:        "Shipment reopened",
		Metadata: map[string]any{
			"proNumber": sh.ProNumber,
		},
		Actor: actor,
	}
}

// BuildOwnershipTransferred emits an ownership transfer event.
func BuildOwnershipTransferred(
	tenant TenantRef,
	sh *shipment.Shipment,
	previousOwnerID, newOwnerID pulid.ID,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     sh.ID,
		Type:           shipmentevent.TypeOwnershipTransferred,
		Severity:       shipmentevent.SeverityMuted,
		Summary:        "Ownership transferred",
		Metadata: map[string]any{
			"proNumber":       sh.ProNumber,
			"previousOwnerId": previousOwnerID.String(),
			"newOwnerId":      newOwnerID.String(),
		},
		Actor: actor,
	}
}

// BuildMoveStatusChanged emits a move-level status transition event.
// Maps to specialized event types when the new status implies a domain milestone.
func BuildMoveStatusChanged(
	tenant TenantRef,
	move *shipment.ShipmentMove,
	previous shipment.MoveStatus,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	eventType, severity, summary := classifyMoveStatus(move.Status)
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     move.ShipmentID,
		MoveID:         move.ID,
		Type:           eventType,
		Severity:       severity,
		Summary:        summary,
		Metadata: map[string]any{
			"previousStatus": string(previous),
			"newStatus":      string(move.Status),
		},
		Actor: actor,
	}
}

// BuildDriverAssigned emits a new assignment event.
func BuildDriverAssigned(
	tenant TenantRef,
	ref AssignmentRef,
	assignment *shipment.Assignment,
	driverName string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     ref.ShipmentID,
		MoveID:         ref.MoveID,
		AssignmentID:   ref.AssignmentID,
		Type:           shipmentevent.TypeDriverAssigned,
		Severity:       shipmentevent.SeverityMuted,
		Summary:        "Driver assigned",
		Metadata:       assignmentMetadata(assignment, driverName),
		Actor:          actor,
	}
}

// BuildDriverReassigned emits a reassignment event.
func BuildDriverReassigned(
	tenant TenantRef,
	ref AssignmentRef,
	assignment *shipment.Assignment,
	driverName string,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     ref.ShipmentID,
		MoveID:         ref.MoveID,
		AssignmentID:   ref.AssignmentID,
		Type:           shipmentevent.TypeDriverReassigned,
		Severity:       shipmentevent.SeverityMuted,
		Summary:        "Driver reassigned",
		Metadata:       assignmentMetadata(assignment, driverName),
		Actor:          actor,
	}
}

// BuildDriverUnassigned emits an unassignment event.
func BuildDriverUnassigned(
	tenant TenantRef,
	ref AssignmentRef,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     ref.ShipmentID,
		MoveID:         ref.MoveID,
		AssignmentID:   ref.AssignmentID,
		Type:           shipmentevent.TypeDriverUnassigned,
		Severity:       shipmentevent.SeverityMuted,
		Summary:        "Driver unassigned",
		Actor:          actor,
	}
}

// BuildHoldPlaced emits a hold creation event.
func BuildHoldPlaced(
	tenant TenantRef,
	hold *shipment.ShipmentHold,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     hold.ShipmentID,
		HoldID:         hold.ID,
		Type:           shipmentevent.TypeHoldPlaced,
		Severity:       shipmentevent.SeverityDanger,
		Summary:        "Hold placed",
		Metadata: map[string]any{
			"holdType":     string(hold.Type),
			"holdSeverity": string(hold.Severity),
			"holdSource":   string(hold.Source),
		},
		Actor: actor,
	}
}

// BuildHoldUpdated emits a hold update event.
func BuildHoldUpdated(
	tenant TenantRef,
	hold *shipment.ShipmentHold,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     hold.ShipmentID,
		HoldID:         hold.ID,
		Type:           shipmentevent.TypeHoldUpdated,
		Severity:       shipmentevent.SeverityMuted,
		Summary:        "Hold updated",
		Metadata: map[string]any{
			"holdType": string(hold.Type),
		},
		Actor: actor,
	}
}

// BuildHoldReleased emits a hold release event.
func BuildHoldReleased(
	tenant TenantRef,
	hold *shipment.ShipmentHold,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     hold.ShipmentID,
		HoldID:         hold.ID,
		Type:           shipmentevent.TypeHoldReleased,
		Severity:       shipmentevent.SeveritySuccess,
		Summary:        "Hold released",
		Metadata: map[string]any{
			"holdType": string(hold.Type),
		},
		Actor: actor,
	}
}

// BuildCommentPosted emits a comment-posted event.
// The full comment body is carried in metadata so the frontend can render the
// summary as "{actor} added a comment to #{pro}" with the body as a detail line.
func BuildCommentPosted(
	tenant TenantRef,
	comment *shipment.ShipmentComment,
	actor services.AuditActor,
) *services.RecordShipmentEventParams {
	mentionIDs := make([]string, 0, len(comment.MentionedUserIDs))
	for _, id := range comment.MentionedUserIDs {
		mentionIDs = append(mentionIDs, id.String())
	}

	return &services.RecordShipmentEventParams{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		ShipmentID:     comment.ShipmentID,
		CommentID:      comment.ID,
		Type:           shipmentevent.TypeCommentPosted,
		Severity:       shipmentevent.SeverityInfo,
		Summary:        "Comment added",
		Metadata: map[string]any{
			"commentBody":       comment.Comment,
			"commentType":       string(comment.Type),
			"commentVisibility": string(comment.Visibility),
			"commentPriority":   string(comment.Priority),
			"mentionedUserIds":  mentionIDs,
		},
		Actor: actor,
	}
}

// classifyMoveStatus translates a move status transition to a specific event
// type, severity, and short subject summary.
func classifyMoveStatus(
	status shipment.MoveStatus,
) (shipmentevent.Type, shipmentevent.Severity, string) {
	switch status {
	case shipment.MoveStatusInTransit:
		return shipmentevent.TypeMoveDeparted, shipmentevent.SeverityBrand, "Move departed"
	case shipment.MoveStatusCompleted:
		return shipmentevent.TypeMoveArrived, shipmentevent.SeveritySuccess, "Move completed"
	case shipment.MoveStatusCanceled:
		return shipmentevent.TypeMoveStatusChanged, shipmentevent.SeverityDanger, "Move canceled"
	case shipment.MoveStatusNew, shipment.MoveStatusAssigned:
		return shipmentevent.TypeMoveStatusChanged, shipmentevent.SeverityMuted, "Move status updated"
	}
	return shipmentevent.TypeMoveStatusChanged, shipmentevent.SeverityMuted, "Move status updated"
}

// severityForShipmentStatus picks a feed severity for a shipment status value.
func severityForShipmentStatus(status shipment.Status) shipmentevent.Severity {
	switch status {
	case shipment.StatusCanceled, shipment.StatusDelayed:
		return shipmentevent.SeverityDanger
	case shipment.StatusInTransit, shipment.StatusReadyToInvoice:
		return shipmentevent.SeverityBrand
	case shipment.StatusCompleted, shipment.StatusInvoiced:
		return shipmentevent.SeveritySuccess
	case shipment.StatusNew,
		shipment.StatusPartiallyAssigned,
		shipment.StatusAssigned,
		shipment.StatusPartiallyCompleted:
		return shipmentevent.SeverityMuted
	}
	return shipmentevent.SeverityMuted
}

func assignmentMetadata(assignment *shipment.Assignment, driverName string) map[string]any {
	meta := map[string]any{
		"primaryWorkerId":   pulidPtrString(assignment.PrimaryWorkerID),
		"secondaryWorkerId": pulidPtrString(assignment.SecondaryWorkerID),
		"tractorId":         pulidPtrString(assignment.TractorID),
		"trailerId":         pulidPtrString(assignment.TrailerID),
	}
	if driverName != "" {
		meta["driverName"] = driverName
	}
	return meta
}

func pulidPtrString(p *pulid.ID) string {
	if p == nil {
		return ""
	}
	return p.String()
}
