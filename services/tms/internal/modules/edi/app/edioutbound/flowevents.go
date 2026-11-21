package edioutbound

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var flowShipmentEvents = []shipmentevent.Type{
	shipmentevent.TypeStatusChanged,
	shipmentevent.TypeShipmentCanceled,
	shipmentevent.TypeShipmentUncanceled,
	shipmentevent.TypeMoveArrived,
	shipmentevent.TypeMoveDeparted,
	shipmentevent.TypeStopArrived,
	shipmentevent.TypeStopDeparted,
	shipmentevent.TypeDriverAssigned,
	shipmentevent.TypeDriverUnassigned,
	shipmentevent.TypeCarrierAssigned,
	shipmentevent.TypeCarrierUnassigned,
}

func FlowShipmentEvent(eventType shipmentevent.Type) bool {
	return slices.Contains(flowShipmentEvents, eventType)
}

func statusShipmentEvent(event *shipmentevent.Event) bool {
	//nolint:exhaustive // Only arrivals and departures answer with a status.
	switch event.Type {
	case shipmentevent.TypeStopArrived, shipmentevent.TypeStopDeparted:
		return true
	case shipmentevent.TypeMoveArrived, shipmentevent.TypeMoveDeparted:
		return !event.FromStopActual()
	}
	return false
}

func dispatchShipmentEvent(event *shipmentevent.Event) bool {
	//nolint:exhaustive // Only a committed driver or carrier dispatches a load.
	switch event.Type {
	case shipmentevent.TypeDriverAssigned, shipmentevent.TypeCarrierAssigned:
		return true
	case shipmentevent.TypeStatusChanged:
		newStatus, _ := event.Metadata["newStatus"].(string)
		status := shipmentvocab.Status(newStatus)
		return status == shipmentvocab.StatusAssigned ||
			status == shipmentvocab.StatusPartiallyAssigned
	}
	return false
}

func firstDispatch(previous, current *lading.LoadState) bool {
	return current.State == lading.LoadDispatched &&
		(!previous.Exists || previous.State == "" || previous.State == lading.LoadOpen)
}

// HasShipmentFlows is whether any partner's flow follows the shipment.
func (s *Service) HasShipmentFlows(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (bool, error) {
	instances, err := s.flowRepo.ListOpenInstances(
		ctx,
		&ladingports.ListShipmentFlowInstancesRequest{
			TenantInfo: tenantInfo,
			ShipmentID: shipmentID,
		},
	)
	if err != nil {
		return false, err
	}
	return len(instances) > 0, nil
}

type NotifyShipmentEventRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	EventID    pulid.ID              `json:"eventId"`
}

// NotifyShipmentEvent tells every flow that follows a shipment what happened to it: the load's
// state, and, for an arrival, departure or delivery, the event a flow may answer with a 214.
func (s *Service) NotifyShipmentEvent(ctx context.Context, req *NotifyShipmentEventRequest) error {
	event, err := s.shipmentEventRepo.GetByID(ctx, shipmenteventports.GetShipmentEventByIDRequest{
		ID:         req.EventID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	instances, err := s.flowRepo.ListOpenInstances(
		ctx,
		&ladingports.ListShipmentFlowInstancesRequest{
			TenantInfo: req.TenantInfo,
			ShipmentID: event.ShipmentID,
		},
	)
	if err != nil || len(instances) == 0 {
		return err
	}
	load, err := s.loadState(ctx, req.TenantInfo, event.ShipmentID)
	if err != nil {
		return err
	}

	var failures []error
	notified := map[pulid.ID]bool{}
	for _, instance := range instances {
		previous, stateErr := flowLoad(instance)
		if stateErr != nil {
			failures = append(failures, stateErr)
			continue
		}
		if err = s.tellLoad(ctx, req.TenantInfo, instance, previous, load); err != nil {
			failures = append(failures, err)
		}
		if notified[instance.EDIPartnerID] {
			continue
		}
		idempotencyKey := ""
		switch {
		case statusShipmentEvent(event):
			idempotencyKey = fmt.Sprintf("flow-214:%s:%s", instance.EDIPartnerID, event.ID)
		case dispatchShipmentEvent(event) && firstDispatch(previous, load):
			idempotencyKey = fmt.Sprintf(
				"flow-dispatched:%s:%s",
				instance.EDIPartnerID,
				event.ShipmentID,
			)
		default:
			continue
		}
		notified[instance.EDIPartnerID] = true
		if err = s.sendShipmentStatus(
			ctx,
			req.TenantInfo,
			instance,
			event,
			idempotencyKey,
		); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func flowLoad(instance *ladingproject.FlowInstance) (*lading.LoadState, error) {
	var summary lading.FlowSummary
	if len(instance.State) > 0 {
		if err := sonic.Unmarshal(instance.State, &summary); err != nil {
			return nil, fmt.Errorf("read flow state: %w", err)
		}
	}
	return &summary.Load, nil
}

func (s *Service) loadState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*lading.LoadState, error) {
	source, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	return &lading.LoadState{Exists: true, State: LoadStateOf(source.Status)}, nil
}

// LoadStateOf is the state a flow sees a load in, from its shipment's status.
func LoadStateOf(status shipmentvocab.Status) string {
	switch status {
	case shipmentvocab.StatusPartiallyAssigned,
		shipmentvocab.StatusAssigned,
		shipmentvocab.StatusInTransit,
		shipmentvocab.StatusDelayed,
		shipmentvocab.StatusPartiallyCompleted:
		return lading.LoadDispatched
	case shipmentvocab.StatusReadyToInvoice, shipmentvocab.StatusCompleted:
		return lading.LoadDelivered
	case shipmentvocab.StatusInvoiced:
		return lading.LoadInvoiced
	case shipmentvocab.StatusCanceled:
		return lading.LoadCancelled
	case shipmentvocab.StatusNew:
	}
	return lading.LoadOpen
}

func (s *Service) tellLoad(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	instance *ladingproject.FlowInstance,
	previous *lading.LoadState,
	load *lading.LoadState,
) error {
	if previous.Exists == load.Exists && previous.State == load.State {
		return nil
	}
	dispatched, err := s.flows.DispatchTo(ctx, instance, &edilading.DispatchRequest{
		TenantInfo: tenantInfo,
		Event:      lading.FlowEvent{Kind: lading.EventLoad, Load: load},
		Now:        time.Now(),
	})
	if err != nil {
		return err
	}
	_, err = s.ExecuteFlowActions(ctx, tenantInfo, dispatched, nil)
	return err
}

// sendShipmentStatus gives a status event to the partner's flow when one of its send clauses
// answers it; a partner whose flow sends nothing on it is left alone.
func (s *Service) sendShipmentStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	instance *ladingproject.FlowInstance,
	event *shipmentevent.Event,
	idempotencyKey string,
) error {
	request := &GenerateEDIDocumentRequest{
		TenantInfo:      tenantInfo,
		EDIPartnerID:    instance.EDIPartnerID,
		ShipmentID:      event.ShipmentID,
		ShipmentEventID: event.ID,
		TransactionSet:  edidocument.TransactionSet214,
		Direction:       edidocument.DocumentDirectionOutbound,
		IdempotencyKey:  idempotencyKey,
	}
	o, err := s.resolveOutbound(ctx, previewRequest(request))
	if err != nil {
		s.l.Info("no 214 for a shipment event",
			zap.String("eventId", event.ID.String()),
			zap.String("partnerId", instance.EDIPartnerID.String()),
			zap.Error(err),
		)
		return nil
	}
	if o.clause == nil || o.clause.Kind != lading.ClauseSend {
		return nil
	}
	if _, err = s.GenerateDocument(ctx, request); err != nil &&
		!errors.Is(err, ediports.ErrSendDeferred) {
		return err
	}
	return nil
}

// attachFlowShipment records the shipment an approved tender became on the flows that follow
// the tender, and tells them the load now exists.
func (s *Service) AttachFlowShipment(ctx context.Context, transfer *editransfer.EDITransfer) {
	tenantInfo := pagination.TenantInfo{
		OrgID: transfer.TargetOrganizationID,
		BuID:  transfer.TargetBusinessUnitID,
	}
	instances, err := s.flowRepo.ListOpenInstances(
		ctx,
		&ladingports.ListShipmentFlowInstancesRequest{
			TenantInfo: tenantInfo,
			TransferID: transfer.ID,
		},
	)
	if err != nil {
		s.l.Warn("failed to find the flows of an approved tender",
			zap.String("transferId", transfer.ID.String()),
			zap.Error(err),
		)
		return
	}
	for _, instance := range instances {
		attached, attachErr := s.flows.Attach(ctx, instance, "", transfer.TargetShipmentID)
		var previous *lading.LoadState
		if attachErr == nil {
			previous, attachErr = flowLoad(attached)
		}
		if attachErr == nil {
			attachErr = s.tellLoad(ctx, tenantInfo, attached, previous, &lading.LoadState{
				Exists: true,
				State:  lading.LoadOpen,
			})
		}
		if attachErr != nil {
			s.l.Warn("failed to give a flow its shipment",
				zap.String("instanceId", instance.ID.String()),
				zap.String("shipmentId", transfer.TargetShipmentID.String()),
				zap.Error(attachErr),
			)
		}
	}
}
