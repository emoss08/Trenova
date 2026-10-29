package internaledistatussync

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/internaledilifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	DB                 kernelports.Transactor
	ShipmentRepo       shipmentports.ShipmentRepository
	ShipmentEventRepo  shipmenteventports.ShipmentEventRepository
	ShipmentLinkRepo   editenderports.EDIShipmentLinkRepository
	TransferChangeRepo editenderports.EDITransferChangeRepository
	Realtime           realtimeports.RealtimeService
	Invalidator        shipmentinvalidationports.ShipmentInvalidator
	Coordinator        *shipmentstate.Coordinator
	OrderDerivation    orderports.OrderDerivationService
}

type Observer struct {
	l                  *zap.Logger
	db                 kernelports.Transactor
	shipmentRepo       shipmentRepository
	shipmentEventRepo  shipmentEventRepository
	shipmentLinkRepo   shipmentLinkRepository
	transferChangeRepo transferChangeRepository
	realtime           realtimeports.RealtimeService
	invalidator        shipmentinvalidationports.ShipmentInvalidator
	orderDerivation    orderports.OrderDerivationService
	lifecycleApplier   *internaledilifecycle.Applier
}

type shipmentRepository interface {
	GetByID(
		ctx context.Context,
		req *shipmentports.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
	UpdateOperationalLifecycle(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*shipment.Shipment, error)
	UpdateStatus(
		ctx context.Context,
		req *shipmentports.UpdateShipmentStatusRequest,
	) (*shipment.Shipment, error)
	Cancel(
		ctx context.Context,
		req *shipmentports.CancelShipmentRequest,
	) (*shipment.Shipment, error)
}

type shipmentEventRepository interface {
	Insert(ctx context.Context, entity *shipmentevent.Event) error
}

type shipmentLinkRepository interface {
	GetShipmentLinksByShipmentID(
		ctx context.Context,
		req editenderports.GetEDIShipmentLinksByShipmentIDRequest,
	) ([]*edisync.ShipmentLink, error)
}

type transferChangeRepository interface {
	CreateTransferChangeIdempotent(
		ctx context.Context,
		entity *edisync.TransferChange,
	) (*editenderports.CreateEDITransferChangeIdempotentResult, error)
}

type side struct {
	organizationID pulid.ID
	shipmentID     pulid.ID
}

type syncContext struct {
	link          *edisync.ShipmentLink
	direction     edisync.TransferChangeDirection
	source        *shipment.Shipment
	target        *shipment.Shipment
	changed       *shipment.Shipment
	opposite      *shipment.Shipment
	oppositeSide  side
	previous      shipmentvocab.Status
	next          shipmentvocab.Status
	changeType    string
	cancelReason  string
	canceledByID  pulid.ID
	transfer      *edisync.TransferChange
	mirroredEvent *shipmentevent.Event
	updated       *shipment.Shipment
	lifecyclePlan *internaledilifecycle.Plan
}

func New(p Params) *Observer {
	return &Observer{
		l:                  p.Logger.Named("service.internal-edi-status-sync"),
		db:                 p.DB,
		shipmentRepo:       p.ShipmentRepo,
		shipmentEventRepo:  p.ShipmentEventRepo,
		shipmentLinkRepo:   p.ShipmentLinkRepo,
		transferChangeRepo: p.TransferChangeRepo,
		realtime:           p.Realtime,
		invalidator:        p.Invalidator,
		orderDerivation:    p.OrderDerivation,
		lifecycleApplier: internaledilifecycle.New(internaledilifecycle.Params{
			ShipmentRepo: p.ShipmentRepo,
			Coordinator:  p.Coordinator,
		}),
	}
}

func (o *Observer) OnShipmentEvent(ctx context.Context, event *shipmentevent.Event) error {
	if event == nil ||
		hasMirroredSource(event) {
		return nil
	}

	if !eventCanSync(event) {
		return nil
	}

	links, err := o.shipmentLinkRepo.GetShipmentLinksByShipmentID(
		ctx,
		editenderports.GetEDIShipmentLinksByShipmentIDRequest{
			ShipmentID: event.ShipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: event.OrganizationID,
				BuID:  event.BusinessUnitID,
			},
		},
	)
	if err != nil {
		return err
	}

	errs := make([]error, 0)
	for _, link := range links {
		if err = o.handleLink(ctx, event, link); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (o *Observer) handleLink(
	ctx context.Context,
	event *shipmentevent.Event,
	link *edisync.ShipmentLink,
) error {
	if link == nil {
		return nil
	}

	var result syncContext
	err := o.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		result, txErr = o.buildSyncContext(txCtx, event, link)
		if txErr != nil || result.link == nil {
			return txErr
		}

		inserted, txErr := o.transferChangeRepo.CreateTransferChangeIdempotent(
			txCtx,
			result.transfer,
		)
		if txErr != nil {
			return txErr
		}
		result.transfer = inserted.TransferChange
		if !inserted.Created || result.transfer.Status != edisync.TransferChangeStatusApplied {
			return nil
		}

		result.updated, txErr = o.applyChange(txCtx, result)
		if txErr != nil {
			return txErr
		}

		if result.updated != nil && o.orderDerivation != nil {
			if txErr = o.orderDerivation.RecomputeForShipment(
				txCtx,
				pagination.TenantInfo{
					OrgID: result.updated.OrganizationID,
					BuID:  result.updated.BusinessUnitID,
				},
				result.updated.ID,
			); txErr != nil {
				return txErr
			}
		}

		result.mirroredEvent = buildMirroredEvent(event, result)
		return o.shipmentEventRepo.Insert(txCtx, result.mirroredEvent)
	})
	if err != nil {
		return err
	}

	o.publishResult(ctx, result)
	return nil
}

func (o *Observer) buildSyncContext(
	ctx context.Context,
	event *shipmentevent.Event,
	link *edisync.ShipmentLink,
) (syncContext, error) {
	result, ok := resolveDirection(event, link)
	if !ok {
		return syncContext{}, nil
	}
	result.link = link
	result.previous = shipmentvocab.Status(metadataString(event.Metadata, "previousStatus"))
	result.cancelReason = metadataString(event.Metadata, "reason")
	result.canceledByID = canceledByIDForEvent(event)

	source, target, err := o.getLinkedShipments(ctx, link)
	if err != nil {
		return syncContext{}, err
	}
	result.source = source
	result.target = target

	if result.direction == edisync.TransferChangeDirectionSourceToTarget {
		result.changed = source
		result.opposite = target
	} else {
		result.changed = target
		result.opposite = source
	}

	if isLifecycleCandidate(event) {
		plan, lifecycleErr := o.lifecycleApplier.PrepareLoaded(
			internaledilifecycle.PrepareLoadedRequest{
				Link:      link,
				Direction: result.direction,
				Source:    source,
				Target:    target,
			},
		)
		if lifecycleErr != nil {
			return syncContext{}, lifecycleErr
		}
		if plan != nil && (len(plan.Diffs) > 0 || len(plan.Conflicts) > 0) {
			result.lifecyclePlan = plan
			result.changeType = edisync.TransferChangeTypeShipmentLifecycle214
			result.next = plan.OppositePrepared.Status
			result.transfer = buildTransferChange(event, result)
			prepareTransferChange(result.transfer, &result)
			return result, nil
		}
		if event.Type != shipmentevent.TypeStatusChanged {
			return syncContext{}, nil
		}
	}

	next, ok := nextStatusForEvent(event)
	if !ok {
		return syncContext{}, nil
	}
	result.next = next
	result.changeType = changeTypeForEvent(event)
	result.transfer = buildTransferChange(event, result)
	prepareTransferChange(result.transfer, &result)
	return result, nil
}

func (o *Observer) getLinkedShipments(
	ctx context.Context,
	link *edisync.ShipmentLink,
) (*shipment.Shipment, *shipment.Shipment, error) {
	source, err := o.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: link.SourceShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: link.SourceOrganizationID,
			BuID:  link.BusinessUnitID,
		},
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	target, err := o.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: link.TargetShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: link.TargetOrganizationID,
			BuID:  link.BusinessUnitID,
		},
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return source, target, nil
}

func (o *Observer) applyChange(
	ctx context.Context,
	result syncContext,
) (*shipment.Shipment, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID: result.oppositeSide.organizationID,
		BuID:  result.link.BusinessUnitID,
	}

	if result.changeType == edisync.TransferChangeTypeShipmentCancel214 {
		return o.shipmentRepo.Cancel(ctx, &shipmentports.CancelShipmentRequest{
			TenantInfo:      tenantInfo,
			ShipmentID:      result.oppositeSide.shipmentID,
			CanceledByID:    result.canceledByID,
			CanceledAt:      timeutils.NowUnix(),
			CancelReason:    result.cancelReason,
			ExpectedVersion: result.opposite.Version,
		})
	}

	if result.changeType == edisync.TransferChangeTypeShipmentLifecycle214 {
		return o.lifecycleApplier.ApplyPrepared(ctx, result.lifecyclePlan)
	}

	return o.shipmentRepo.UpdateStatus(
		ctx,
		&shipmentports.UpdateShipmentStatusRequest{
			TenantInfo: tenantInfo,
			ShipmentID: result.oppositeSide.shipmentID,
			Status:     result.next,
			Version:    result.opposite.Version,
		},
	)
}

func (o *Observer) publishResult(ctx context.Context, result syncContext) {
	if result.updated != nil {
		shipmentinvalidationports.InvalidateShipments(
			ctx,
			o.invalidator,
			&shipmentinvalidationports.ShipmentInvalidation{
				OrganizationID: result.updated.OrganizationID,
				BusinessUnitID: result.updated.BusinessUnitID,
				Action:         "updated",
				RecordID:       result.updated.ID,
				Entity:         result.updated,
			},
		)
	}
	if result.mirroredEvent != nil {
		if err := realtimeinvalidation.Publish(
			ctx,
			o.realtime,
			&realtimeinvalidation.PublishParams{
				OrganizationID: result.mirroredEvent.OrganizationID,
				BusinessUnitID: result.mirroredEvent.BusinessUnitID,
				Resource:       "shipmentEvents",
				Action:         "created",
				RecordID:       result.mirroredEvent.ID,
				Entity:         result.mirroredEvent,
			},
		); err != nil {
			o.l.Warn("failed to publish mirrored shipment event invalidation", zap.Error(err))
		}
	}
	if result.transfer != nil {
		if err := realtimeinvalidation.Publish(
			ctx,
			o.realtime,
			&realtimeinvalidation.PublishParams{
				OrganizationID: result.oppositeSide.organizationID,
				BusinessUnitID: result.transfer.BusinessUnitID,
				Resource:       "ediTransferChanges",
				Action:         "created",
				RecordID:       result.transfer.ID,
				Entity:         result.transfer,
			},
		); err != nil {
			o.l.Warn("failed to publish EDI transfer change invalidation", zap.Error(err))
		}
	}
}

func resolveDirection(
	event *shipmentevent.Event,
	link *edisync.ShipmentLink,
) (syncContext, bool) {
	if event.OrganizationID == link.SourceOrganizationID &&
		event.ShipmentID == link.SourceShipmentID {
		return syncContext{
			direction: edisync.TransferChangeDirectionSourceToTarget,
			oppositeSide: side{
				organizationID: link.TargetOrganizationID,
				shipmentID:     link.TargetShipmentID,
			},
		}, true
	}
	if event.OrganizationID == link.TargetOrganizationID &&
		event.ShipmentID == link.TargetShipmentID {
		return syncContext{
			direction: edisync.TransferChangeDirectionTargetToSource,
			oppositeSide: side{
				organizationID: link.SourceOrganizationID,
				shipmentID:     link.SourceShipmentID,
			},
		}, true
	}
	return syncContext{}, false
}

func buildTransferChange(
	event *shipmentevent.Event,
	result syncContext,
) *edisync.TransferChange {
	idempotencyKey := fmt.Sprintf(
		"%s:%s:%s:%s",
		result.link.ID,
		event.ID,
		result.direction,
		result.changeType,
	)
	payload := buildPayload(event, result)

	diff := map[string]any{
		"field":          "status",
		"previousStatus": string(result.opposite.Status),
		"newStatus":      string(result.next),
	}
	if result.changeType == edisync.TransferChangeTypeShipmentLifecycle214 &&
		result.lifecyclePlan != nil {
		diff = map[string]any{
			"field":          "stopActuals",
			"previousStatus": string(result.lifecyclePlan.OppositeOriginal.Status),
			"newStatus":      string(result.lifecyclePlan.OppositePrepared.Status),
			"stops":          result.lifecyclePlan.Diffs,
		}
	}

	return &edisync.TransferChange{
		BusinessUnitID:        result.link.BusinessUnitID,
		ShipmentLinkID:        result.link.ID,
		Direction:             result.direction,
		ChangeType:            result.changeType,
		IdempotencyKey:        idempotencyKey,
		SourceShipmentVersion: result.source.Version,
		TargetShipmentVersion: result.target.Version,
		Payload:               payload,
		Diff:                  diff,
	}
}

func prepareTransferChange(change *edisync.TransferChange, result *syncContext) {
	change.Status = edisync.TransferChangeStatusApplied
	change.ConflictStatus = edisync.TransferChangeConflictNone
	now := timeutils.NowUnix()
	change.AppliedAt = &now

	var cancelBlocker error
	if result.changeType == edisync.TransferChangeTypeShipmentCancel214 {
		cancelBlocker = shipmentstate.ValidateCancel(result.opposite)
	}

	switch {
	case result.link.Status != edisync.ShipmentLinkStatusActive:
		change.Status = edisync.TransferChangeStatusIgnored
		change.AppliedAt = nil
		change.Payload["ignoredReason"] = "shipment link is not active"
	case result.link.SyncPolicy == edisync.ShipmentSyncPolicyReadOnly:
		change.Status = edisync.TransferChangeStatusIgnored
		change.AppliedAt = nil
		change.Payload["ignoredReason"] = "shipment link is read-only"
	case result.opposite.Status == result.next &&
		result.changeType != edisync.TransferChangeTypeShipmentLifecycle214:
		change.Status = edisync.TransferChangeStatusIgnored
		change.AppliedAt = nil
		change.Payload["ignoredReason"] = ignoredSameStatusReason(*result)
	case result.changeType == edisync.TransferChangeTypeShipmentLifecycle214 &&
		result.lifecyclePlan != nil &&
		result.lifecyclePlan.ConflictReason != "":
		change.Status = edisync.TransferChangeStatusPendingReview
		change.ConflictStatus = edisync.TransferChangeConflictConflict
		change.ConflictReason = result.lifecyclePlan.ConflictReason
		change.AppliedAt = nil
		change.Payload["conflictReason"] = change.ConflictReason
		change.Payload["conflicts"] = result.lifecyclePlan.Conflicts
	case cancelBlocker != nil:
		change.Status = edisync.TransferChangeStatusPendingReview
		change.ConflictStatus = edisync.TransferChangeConflictConflict
		change.ConflictReason = "Cannot cancel linked shipment: " + cancelBlocker.Error()
		change.AppliedAt = nil
		change.Payload["conflictReason"] = change.ConflictReason
	case result.changeType != edisync.TransferChangeTypeShipmentLifecycle214 &&
		!shipmentstate.CanTransitionShipmentStatus(result.opposite.Status, result.next):
		change.Status = edisync.TransferChangeStatusPendingReview
		change.ConflictStatus = edisync.TransferChangeConflictConflict
		change.ConflictReason = "Cannot transition linked shipment from " +
			string(result.opposite.Status) + " to " + string(result.next)
		change.AppliedAt = nil
		change.Payload["conflictReason"] = change.ConflictReason
	case result.link.SyncPolicy == edisync.ShipmentSyncPolicyManualReview:
		change.Status = edisync.TransferChangeStatusPendingReview
		change.AppliedAt = nil
		change.Payload["reviewReason"] = "shipment link requires manual review"
	case !canAutoApply(result.link.SyncPolicy):
		change.Status = edisync.TransferChangeStatusIgnored
		change.AppliedAt = nil
		change.Payload["ignoredReason"] = "shipment link sync policy does not auto-apply status"
	}
}

func buildPayload(event *shipmentevent.Event, result syncContext) map[string]any {
	previousStatus := result.opposite.Status
	newStatus := result.next
	if result.changeType == edisync.TransferChangeTypeShipmentLifecycle214 &&
		result.lifecyclePlan != nil {
		previousStatus = result.lifecyclePlan.OppositeOriginal.Status
		newStatus = result.lifecyclePlan.OppositePrepared.Status
	}

	statusPayload := edipayload.ShipmentStatusPayload{
		ShipmentID: result.changed.ID,
		BOL:        result.changed.BOL,
		ProNumber:  result.changed.ProNumber,
		StatusCode: shipmentStatusCode(newStatus),
		EventDate:  event.OccurredAt,
		EventTime:  event.OccurredAt,
		References: map[string]string{
			"eventId":          event.ID.String(),
			"eventType":        string(event.Type),
			"shipmentId":       result.changed.ID.String(),
			"sourceShipmentId": result.link.SourceShipmentID.String(),
			"targetShipmentId": result.link.TargetShipmentID.String(),
			"sourceOrgId":      result.link.SourceOrganizationID.String(),
			"targetOrgId":      result.link.TargetOrganizationID.String(),
			"previousStatus":   string(result.previous),
			"newStatus":        string(newStatus),
		},
	}

	payload := map[string]any{
		"transactionSet": string(edidocument.TransactionSet214),
		"shipmentStatus": statusPayload,
		"sourceEventId":  event.ID.String(),
		"shipmentLinkId": result.link.ID.String(),
		"direction":      string(result.direction),
		"sourceShipment": map[string]any{
			"id":             result.link.SourceShipmentID.String(),
			"organizationId": result.link.SourceOrganizationID.String(),
			"status":         string(result.source.Status),
		},
		"targetShipment": map[string]any{
			"id":             result.link.TargetShipmentID.String(),
			"organizationId": result.link.TargetOrganizationID.String(),
			"status":         string(result.target.Status),
		},
		"previousStatus": string(previousStatus),
		"newStatus":      string(newStatus),
	}
	if result.changeType == edisync.TransferChangeTypeShipmentCancel214 {
		payload["cancellationReason"] = result.cancelReason
	}
	if result.changeType == edisync.TransferChangeTypeShipmentLifecycle214 &&
		result.lifecyclePlan != nil {
		payload["matchedStopActualDiffs"] = result.lifecyclePlan.Diffs
	}

	return payload
}

func buildMirroredEvent(
	sourceEvent *shipmentevent.Event,
	result syncContext,
) *shipmentevent.Event {
	eventType := shipmentevent.TypeStatusChanged
	summary := "Status synced from internal EDI 214"
	previousStatus := result.opposite.Status
	newStatus := result.next
	if result.changeType == edisync.TransferChangeTypeShipmentLifecycle214 &&
		result.lifecyclePlan != nil {
		summary = "Lifecycle synced from internal EDI 214"
		previousStatus = result.lifecyclePlan.OppositeOriginal.Status
		newStatus = result.updated.Status
	}
	metadata := map[string]any{
		"proNumber":      result.updated.ProNumber,
		"previousStatus": string(previousStatus),
		"newStatus":      string(newStatus),
		edisync.InternalEDIMirroredFromEventIDKey: sourceEvent.ID.String(),
		edisync.InternalEDIShipmentLinkIDKey:      result.link.ID.String(),
	}
	if result.changeType == edisync.TransferChangeTypeShipmentCancel214 {
		eventType = shipmentevent.TypeShipmentCanceled
		summary = "Cancellation synced from internal EDI 214"
		metadata["reason"] = result.cancelReason
	}

	return &shipmentevent.Event{
		OrganizationID: result.updated.OrganizationID,
		BusinessUnitID: result.updated.BusinessUnitID,
		ShipmentID:     result.updated.ID,
		Type:           eventType,
		Severity:       severityForStatus(newStatus),
		ActorType:      shipmentevent.ActorEDI,
		ActorLabel:     "Internal EDI",
		Summary:        summary,
		Metadata:       metadata,
		OccurredAt:     sourceEvent.OccurredAt,
		CorrelationID:  sourceEvent.CorrelationID,
	}
}

func nextStatusForEvent(event *shipmentevent.Event) (shipmentvocab.Status, bool) {
	switch event.Type {
	case shipmentevent.TypeStatusChanged:
		next := shipmentvocab.Status(metadataString(event.Metadata, "newStatus"))
		return next, next != ""
	case shipmentevent.TypeShipmentCanceled:
		return shipmentvocab.StatusCanceled, true
	default:
		return "", false
	}
}

func eventCanSync(event *shipmentevent.Event) bool {
	if event == nil {
		return false
	}

	if _, ok := nextStatusForEvent(event); ok {
		return true
	}

	return isLifecycleCandidate(event)
}

func isLifecycleCandidate(event *shipmentevent.Event) bool {
	switch event.Type {
	case shipmentevent.TypeStatusChanged,
		shipmentevent.TypeStopCompleted,
		shipmentevent.TypeMoveStatusChanged,
		shipmentevent.TypeMoveDeparted,
		shipmentevent.TypeMoveArrived:
		return true
	default:
		return false
	}
}

func changeTypeForEvent(event *shipmentevent.Event) string {
	if event.Type == shipmentevent.TypeShipmentCanceled {
		return edisync.TransferChangeTypeShipmentCancel214
	}
	return edisync.TransferChangeTypeShipmentStatus214
}

func ignoredSameStatusReason(result syncContext) string {
	if result.changeType == edisync.TransferChangeTypeShipmentCancel214 {
		return "linked shipment is already canceled"
	}
	return "linked shipment already has status"
}

func canceledByIDForEvent(event *shipmentevent.Event) pulid.ID {
	if event.ActorType == shipmentevent.ActorUser {
		return event.ActorID
	}
	return pulid.Nil
}

func canAutoApply(policy edisync.ShipmentSyncPolicy) bool {
	return policy == edisync.ShipmentSyncPolicyAutoOperational ||
		policy == edisync.ShipmentSyncPolicyAutoAllSafe
}

func hasMirroredSource(event *shipmentevent.Event) bool {
	return metadataString(event.Metadata, edisync.InternalEDIMirroredFromEventIDKey) != ""
}

func metadataString(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	value := metadata[key]
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case []byte:
		return string(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return ""
	}
}

func shipmentStatusCode(status shipmentvocab.Status) string {
	switch status {
	case shipmentvocab.StatusInTransit:
		return "AF"
	case shipmentvocab.StatusCompleted,
		shipmentvocab.StatusReadyToInvoice,
		shipmentvocab.StatusInvoiced:
		return "D1"
	case shipmentvocab.StatusDelayed:
		return "A3"
	case shipmentvocab.StatusCanceled:
		return "A7"
	default:
		return "X3"
	}
}

func severityForStatus(status shipmentvocab.Status) shipmentevent.Severity {
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
