package editransferchangeservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/internaledilifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type approvedTransferChangeContext struct {
	link         *edisync.ShipmentLink
	opposite     *shipment.Shipment
	oppositeInfo pagination.TenantInfo
	oppositeID   pulid.ID
	nextStatus   shipmentvocab.Status
}

func (s *Service) applyApprovedTransferChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	change *edisync.TransferChange,
	actor *permissionports.RequestActor,
	appliedAt int64,
) error {
	switch change.ChangeType {
	case edisync.TransferChangeTypeShipmentStatus214,
		edisync.TransferChangeTypeShipmentCancel214,
		edisync.TransferChangeTypeShipmentLifecycle214:
	default:
		return nil
	}

	if change.ChangeType == edisync.TransferChangeTypeShipmentLifecycle214 {
		return s.applyApprovedLifecycleTransferChange(ctx, tenantInfo, change, appliedAt)
	}

	applyCtx, err := s.buildApprovedTransferChangeContext(ctx, tenantInfo, change)
	if err != nil {
		return err
	}
	if applyCtx.opposite.Status == applyCtx.nextStatus {
		return nil
	}

	var updated *shipment.Shipment
	switch change.ChangeType {
	case edisync.TransferChangeTypeShipmentCancel214:
		if err = shipmentstate.ValidateCancel(applyCtx.opposite); err != nil {
			return err
		}
		updated, err = s.shipmentRepo.Cancel(ctx, &shipmentports.CancelShipmentRequest{
			TenantInfo:      applyCtx.oppositeInfo,
			ShipmentID:      applyCtx.oppositeID,
			CanceledByID:    actor.UserID,
			CanceledAt:      appliedAt,
			CancelReason:    transferChangePayloadString(change.Payload, "cancellationReason"),
			ExpectedVersion: applyCtx.opposite.Version,
		})
	case edisync.TransferChangeTypeShipmentStatus214:
		updated, err = s.shipmentRepo.UpdateStatus(
			ctx,
			&shipmentports.UpdateShipmentStatusRequest{
				TenantInfo: applyCtx.oppositeInfo,
				ShipmentID: applyCtx.oppositeID,
				Status:     applyCtx.nextStatus,
				Version:    applyCtx.opposite.Version,
			},
		)
	}
	if err != nil {
		return err
	}

	if recomputeErr := s.recomputeOrderForUpdatedShipment(ctx, updated); recomputeErr != nil {
		return recomputeErr
	}

	return s.shipmentEventRepo.Insert(ctx, buildApprovedTransferChangeEvent(
		change,
		applyCtx,
		updated,
		appliedAt,
	))
}

func (s *Service) recomputeOrderForUpdatedShipment(
	ctx context.Context,
	updated *shipment.Shipment,
) error {
	if updated == nil || s.orderDerivation == nil {
		return nil
	}

	return s.orderDerivation.RecomputeForShipment(
		ctx,
		pagination.TenantInfo{
			OrgID: updated.OrganizationID,
			BuID:  updated.BusinessUnitID,
		},
		updated.ID,
	)
}

func (s *Service) applyApprovedLifecycleTransferChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	change *edisync.TransferChange,
	appliedAt int64,
) error {
	link, err := s.shipmentLinkRepo.GetShipmentLinkByID(
		ctx,
		editenderports.GetEDIShipmentLinkByIDRequest{
			ID:         change.ShipmentLinkID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return err
	}

	applier := s.lifecycleApplier
	if applier == nil {
		applier = internaledilifecycle.New(internaledilifecycle.Params{
			ShipmentRepo: s.shipmentRepo,
			Coordinator:  s.coordinator,
		})
	}

	plan, err := applier.Prepare(ctx, internaledilifecycle.PrepareRequest{
		Link:      link,
		Direction: change.Direction,
	})
	if err != nil {
		return err
	}
	if plan == nil || len(plan.Diffs) == 0 {
		return nil
	}
	if plan.ConflictReason != "" {
		return fmt.Errorf(
			"EDI transfer change lifecycle validation failed: %s",
			plan.ConflictReason,
		)
	}

	updated, err := applier.ApplyPrepared(ctx, plan)
	if err != nil {
		return err
	}

	if recomputeErr := s.recomputeOrderForUpdatedShipment(ctx, updated); recomputeErr != nil {
		return recomputeErr
	}

	return s.shipmentEventRepo.Insert(ctx, buildApprovedLifecycleTransferChangeEvent(
		change,
		plan,
		updated,
		appliedAt,
	))
}

func (s *Service) buildApprovedTransferChangeContext(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	change *edisync.TransferChange,
) (approvedTransferChangeContext, error) {
	link, err := s.shipmentLinkRepo.GetShipmentLinkByID(
		ctx,
		editenderports.GetEDIShipmentLinkByIDRequest{
			ID:         change.ShipmentLinkID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return approvedTransferChangeContext{}, err
	}

	source, target, err := s.getTransferChangeShipments(ctx, link)
	if err != nil {
		return approvedTransferChangeContext{}, err
	}

	nextStatus := shipmentvocab.Status(transferChangePayloadString(change.Payload, "newStatus"))
	if change.ChangeType == edisync.TransferChangeTypeShipmentCancel214 {
		nextStatus = shipmentvocab.StatusCanceled
	}
	if nextStatus == "" {
		return approvedTransferChangeContext{}, fmt.Errorf(
			"EDI transfer change %s is missing new status",
			change.ID,
		)
	}

	result := approvedTransferChangeContext{
		link:       link,
		nextStatus: nextStatus,
	}
	if change.Direction == edisync.TransferChangeDirectionSourceToTarget {
		result.opposite = target
		result.oppositeID = link.TargetShipmentID
		result.oppositeInfo = pagination.TenantInfo{
			OrgID: link.TargetOrganizationID,
			BuID:  link.BusinessUnitID,
		}
		return result, nil
	}

	result.opposite = source
	result.oppositeID = link.SourceShipmentID
	result.oppositeInfo = pagination.TenantInfo{
		OrgID: link.SourceOrganizationID,
		BuID:  link.BusinessUnitID,
	}
	return result, nil
}

func (s *Service) getTransferChangeShipments(
	ctx context.Context,
	link *edisync.ShipmentLink,
) (*shipment.Shipment, *shipment.Shipment, error) {
	source, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: link.SourceShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: link.SourceOrganizationID,
			BuID:  link.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	target, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: link.TargetShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: link.TargetOrganizationID,
			BuID:  link.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return source, target, nil
}

func buildApprovedTransferChangeEvent(
	change *edisync.TransferChange,
	ctx approvedTransferChangeContext,
	updated *shipment.Shipment,
	occurredAt int64,
) *shipmentevent.Event {
	eventType := shipmentevent.TypeStatusChanged
	summary := "Status synced from internal EDI 214"
	metadata := map[string]any{
		"proNumber":      updated.ProNumber,
		"previousStatus": string(ctx.opposite.Status),
		"newStatus":      string(ctx.nextStatus),
		edisync.InternalEDIMirroredFromEventIDKey: transferChangePayloadString(
			change.Payload,
			"sourceEventId",
		),
		edisync.InternalEDIShipmentLinkIDKey: ctx.link.ID.String(),
	}
	if change.ChangeType == edisync.TransferChangeTypeShipmentCancel214 {
		eventType = shipmentevent.TypeShipmentCanceled
		summary = "Cancellation synced from internal EDI 214"
		metadata["reason"] = transferChangePayloadString(change.Payload, "cancellationReason")
	}

	return &shipmentevent.Event{
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		ShipmentID:     updated.ID,
		Type:           eventType,
		Severity:       transferChangeSeverityForStatus(ctx.nextStatus),
		ActorType:      shipmentevent.ActorEDI,
		ActorLabel:     "Internal EDI",
		Summary:        summary,
		Metadata:       metadata,
		OccurredAt:     occurredAt,
	}
}

func buildApprovedLifecycleTransferChangeEvent(
	change *edisync.TransferChange,
	plan *internaledilifecycle.Plan,
	updated *shipment.Shipment,
	occurredAt int64,
) *shipmentevent.Event {
	return &shipmentevent.Event{
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		ShipmentID:     updated.ID,
		Type:           shipmentevent.TypeStatusChanged,
		Severity:       transferChangeSeverityForStatus(updated.Status),
		ActorType:      shipmentevent.ActorEDI,
		ActorLabel:     "Internal EDI",
		Summary:        "Lifecycle synced from internal EDI 214",
		Metadata: map[string]any{
			"proNumber":              updated.ProNumber,
			"previousStatus":         string(plan.OppositeOriginal.Status),
			"newStatus":              string(updated.Status),
			"matchedStopActualDiffs": plan.Diffs,
			edisync.InternalEDIMirroredFromEventIDKey: transferChangePayloadString(
				change.Payload,
				"sourceEventId",
			),
			edisync.InternalEDIShipmentLinkIDKey: plan.Link.ID.String(),
		},
		OccurredAt: occurredAt,
	}
}

func transferChangePayloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}

	value := payload[key]
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

func transferChangeSeverityForStatus(status shipmentvocab.Status) shipmentevent.Severity {
	switch status {
	case shipmentvocab.StatusCanceled, shipmentvocab.StatusDelayed:
		return shipmentevent.SeverityDanger
	case shipmentvocab.StatusCompleted,
		shipmentvocab.StatusReadyToInvoice,
		shipmentvocab.StatusInvoiced:
		return shipmentevent.SeveritySuccess
	case shipmentvocab.StatusInTransit:
		return shipmentevent.SeverityBrand
	default:
		return shipmentevent.SeverityMuted
	}
}
