package ediservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	EDIAlertEventFlowEscalated = "edi.flow.escalated"
	EDIAlertEventFlowHeld      = "edi.flow.held"
)

type FlowActionContext struct {
	GeneratedByID        pulid.ID
	DisableDeliveryQueue bool
	IdempotencyKey       string
	Payload              *edi.DocumentPayload
	TransferID           pulid.ID
}

// ExecuteFlowActions carries out what a flow step asked of Trenova that is not about a message
// it received: sends, resends, holds and escalations. It returns the messages it sent.
func (s *Service) ExecuteFlowActions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	dispatched *edilading.Dispatched,
	actionCtx *FlowActionContext,
) ([]*edi.EDIMessage, error) {
	if actionCtx == nil {
		actionCtx = &FlowActionContext{}
	}
	sent := make([]*edi.EDIMessage, 0)
	var failures []error
	for index := range dispatched.Actions {
		action := &dispatched.Actions[index]
		switch action.Kind {
		case lading.ActionSend:
			message, err := s.SendFlowMessage(ctx, tenantInfo, dispatched, action, actionCtx)
			if err != nil {
				failures = append(failures, fmt.Errorf("send %s: %w", action.Set, err))
				continue
			}
			sent = append(sent, message)
		case lading.ActionResend:
			if err := s.resendFlowMessage(ctx, tenantInfo, dispatched.Instance, action.ID); err != nil {
				failures = append(failures, fmt.Errorf("resend %s: %w", action.ID, err))
			}
		case lading.ActionEscalate:
			s.alertFlow(ctx, dispatched.Instance, EDIAlertEventFlowEscalated, action.Message)
		case lading.ActionHold:
			s.alertFlow(ctx, dispatched.Instance, EDIAlertEventFlowHeld, action.Message)
		}
	}
	return sent, errors.Join(failures...)
}

// SendFlowMessage writes and keeps the message a flow sends, and tells the flow its control
// number so it can match the partner's acknowledgment.
func (s *Service) SendFlowMessage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	dispatched *edilading.Dispatched,
	action *lading.FlowAction,
	actionCtx *FlowActionContext,
) (*edi.EDIMessage, error) {
	instance := dispatched.Instance
	partner, err := s.partnerRepo.GetByID(ctx, ediports.GetEDIPartnerByIDRequest{
		ID:         instance.EDIPartnerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	set := edi.TransactionSet(action.Set)
	documentType, err := s.documentTypeFor(ctx, set, edi.DocumentDirectionOutbound)
	if err != nil {
		return nil, err
	}
	o := &outbound{
		partner:      partner,
		program:      dispatched.Program,
		set:          set,
		mapName:      action.Map,
		document:     action.Document,
		documentType: documentType,
		shipmentID:   instance.ShipmentID,
		payload:      edi.DocumentPayload{TransactionSet: set},
	}
	if dispatched.Program.Manifest.Map(action.Map) == nil {
		return nil, errortypes.NewBusinessError("The flow sends with map {0}, which the version lacks", action.Map)
	}
	if actionCtx.Payload != nil && actionCtx.Payload.TransactionSet == set {
		o.payload = *actionCtx.Payload
	}
	transferID := actionCtx.TransferID
	if transferID.IsNil() {
		transferID = instance.TransferID
	}
	message, err := s.write(ctx, &GenerateEDIDocumentRequest{
		TenantInfo:           tenantInfo,
		EDIPartnerID:         partner.ID,
		TransactionSet:       set,
		Direction:            edi.DocumentDirectionOutbound,
		ShipmentID:           instance.ShipmentID,
		TransferID:           transferID,
		GeneratedByID:        actionCtx.GeneratedByID,
		DisableDeliveryQueue: actionCtx.DisableDeliveryQueue,
		IdempotencyKey:       flowIdempotencyKey(actionCtx.IdempotencyKey, instance, action.ID),
	}, o, &writeOptions{flowInstanceID: instance.ID, flowSendID: action.ID})
	if err != nil {
		return nil, err
	}
	if _, err = s.flows.DispatchTo(ctx, instance, &edilading.DispatchRequest{
		TenantInfo: tenantInfo,
		Program:    dispatched.Program,
		Event: lading.FlowEvent{
			Kind:    lading.EventSent,
			ID:      action.ID,
			Control: message.TransactionControlNumber,
		},
		MessageID: message.ID,
	}); err != nil {
		s.l.Warn("failed to report a sent message to its flow",
			zap.String("instanceId", instance.ID.String()),
			zap.String("messageId", message.ID.String()),
			zap.Error(err),
		)
	}
	return message, nil
}

func flowIdempotencyKey(given string, instance *edi.FlowInstance, sendID string) string {
	if given != "" {
		return given
	}
	return "flow:" + instance.ID.String() + ":" + sendID
}

func (s *Service) resendFlowMessage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	instance *edi.FlowInstance,
	sendID string,
) error {
	message, err := s.flowRepo.GetSendMessage(ctx, ediports.GetFlowSendMessageRequest{
		TenantInfo: tenantInfo,
		InstanceID: instance.ID,
		SendID:     sendID,
	})
	if err != nil {
		return err
	}
	_, err = s.ReplayMessageDelivery(ctx, &RetryMessageDeliveryRequest{
		MessageID:  message.ID,
		TenantInfo: tenantInfo,
	})
	return err
}

func (s *Service) alertFlow(
	ctx context.Context,
	instance *edi.FlowInstance,
	eventType string,
	reason *wasmmodule.Message,
) {
	title := "EDI flow escalated"
	if eventType == EDIAlertEventFlowHeld {
		title = "EDI flow held"
	}
	text := ""
	if reason != nil {
		text = reason.String()
	}
	s.NotifyOperationalFailure(ctx, &EDIOperationalAlert{
		OrganizationID: instance.OrganizationID,
		BusinessUnitID: instance.BusinessUnitID,
		EventType:      eventType,
		PartnerID:      instance.EDIPartnerID,
		Title:          title,
		Message:        fmt.Sprintf("%s %s: %s", instance.Flow, instance.Key, text),
		RelatedEntities: map[string]any{
			"flowInstanceId": instance.ID,
			"partnerId":      instance.EDIPartnerID,
		},
		Data: map[string]any{
			"flow": instance.Flow,
			"key":  instance.Key,
			"link": "/edi/exceptions?panelType=edit&panelEntityId=" + instance.ID.String(),
		},
	})
}

// DispatchFlowTimer wakes a flow instance whose timer fired and carries out what it asks.
func (s *Service) DispatchFlowTimer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	instanceID pulid.ID,
) (*edi.FlowInstance, error) {
	instance, err := s.flowRepo.GetInstance(ctx, ediports.GetFlowInstanceRequest{
		TenantInfo: tenantInfo,
		ID:         instanceID,
	})
	if err != nil {
		return nil, err
	}
	if instance.WakeAt == nil || *instance.WakeAt > time.Now().Unix() {
		return instance, nil
	}
	dispatched, err := s.flows.DispatchTo(ctx, instance, &edilading.DispatchRequest{
		TenantInfo: tenantInfo,
		Event:      lading.FlowEvent{Kind: lading.EventTimer},
	})
	if err != nil {
		return nil, err
	}
	if _, err = s.ExecuteFlowActions(ctx, tenantInfo, dispatched, nil); err != nil {
		return dispatched.Instance, err
	}
	return dispatched.Instance, nil
}

type NotifyFlowEventRequest struct {
	TenantInfo pagination.TenantInfo
	PartnerID  pulid.ID
	Event      string
	Document   jsonutils.RawJSON
	Key        string
	ShipmentID pulid.ID
}

// NotifyFlowEvent tells the partner's flows that something happened in Trenova; a flow that
// sends on it may write a message now, later with others, or not at all.
func (s *Service) NotifyFlowEvent(
	ctx context.Context,
	req *NotifyFlowEventRequest,
) ([]*edi.EDIMessage, error) {
	program, err := s.lading.Active(ctx, req.TenantInfo, req.PartnerID)
	if err != nil {
		if errors.Is(err, edilading.ErrNoProject) || errors.Is(err, edilading.ErrNoActiveVersion) {
			return nil, nil
		}
		return nil, err
	}
	flow, clause := program.Manifest.Handling(req.Event)
	if flow == nil || clause == nil {
		return nil, nil
	}
	dispatched, err := s.flows.Dispatch(ctx, &edilading.DispatchRequest{
		TenantInfo:  req.TenantInfo,
		PartnerID:   req.PartnerID,
		Program:     program,
		Flow:        flow.Name,
		FallbackKey: req.Key,
		Event: lading.FlowEvent{
			Kind:     lading.EventTrenova,
			Event:    req.Event,
			Document: req.Document,
		},
		ShipmentID: req.ShipmentID,
	})
	if err != nil {
		return nil, err
	}
	return s.ExecuteFlowActions(ctx, req.TenantInfo, dispatched, nil)
}

type DecideTenderRequest struct {
	TenantInfo pagination.TenantInfo
	TransferID pulid.ID
	Accept     bool
	Reason     string
	ActorID    pulid.ID
}

// DecideTender gives a planner's decision on a tender to the flow it arrived in, which answers
// the partner. It reports whether a flow took the decision.
func (s *Service) DecideTender(
	ctx context.Context,
	req *DecideTenderRequest,
) (bool, []*edi.EDIMessage, error) {
	instances, err := s.flowRepo.ListInstances(ctx, &ediports.ListFlowInstancesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: req.TenantInfo,
			Pagination: pagination.Info{Limit: 1},
		},
		TransferID: req.TransferID,
	})
	if err != nil {
		return false, nil, err
	}
	if len(instances.Items) == 0 {
		return false, nil, nil
	}
	instance, err := s.flowRepo.GetInstance(ctx, ediports.GetFlowInstanceRequest{
		TenantInfo: req.TenantInfo,
		ID:         instances.Items[0].ID,
	})
	if err != nil {
		return false, nil, err
	}
	decision := lading.DecisionDecline
	if req.Accept {
		decision = lading.DecisionAccept
	}
	dispatched, err := s.flows.DispatchTo(ctx, instance, &edilading.DispatchRequest{
		TenantInfo: req.TenantInfo,
		Event: lading.FlowEvent{
			Kind:     lading.EventPlanner,
			Decision: decision,
			Reason:   req.Reason,
		},
		ActorID: req.ActorID,
	})
	if err != nil {
		return true, nil, err
	}
	sent, err := s.ExecuteFlowActions(ctx, req.TenantInfo, dispatched, &FlowActionContext{
		GeneratedByID: req.ActorID,
		TransferID:    req.TransferID,
	})
	return true, sent, err
}

// AcknowledgeFlowMessage tells the flow that sent a message how the partner acknowledged it.
func (s *Service) AcknowledgeFlowMessage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	message *edi.EDIMessage,
	accepted bool,
) error {
	if message == nil || message.FlowInstanceID.IsNil() {
		return nil
	}
	instance, err := s.flowRepo.GetInstance(ctx, ediports.GetFlowInstanceRequest{
		TenantInfo: tenantInfo,
		ID:         message.FlowInstanceID,
	})
	if err != nil {
		return err
	}
	dispatched, err := s.flows.DispatchTo(ctx, instance, &edilading.DispatchRequest{
		TenantInfo: tenantInfo,
		Event: lading.FlowEvent{
			Kind:     lading.EventAck,
			Control:  message.TransactionControlNumber,
			Accepted: accepted,
		},
		MessageID: message.ID,
	})
	if err != nil {
		return err
	}
	_, err = s.ExecuteFlowActions(ctx, tenantInfo, dispatched, nil)
	return err
}

func (s *Service) ListFlowInstances(
	ctx context.Context,
	req *ediports.ListFlowInstancesRequest,
) (*pagination.ListResult[*edi.FlowInstance], error) {
	return s.flowRepo.ListInstances(ctx, req)
}

func (s *Service) ListFlowInstancesCursor(
	ctx context.Context,
	req *ediports.ListFlowInstancesRequest,
) (*pagination.CursorListResult[*edi.FlowInstance], error) {
	return s.flowRepo.ListInstancesCursor(ctx, req)
}

type FlowInstanceDetail struct {
	Instance *edi.FlowInstance  `json:"instance"`
	Summary  lading.FlowSummary `json:"summary"`
	Events   []*edi.FlowEvent   `json:"events"`
}

func (s *Service) GetFlowInstance(
	ctx context.Context,
	req ediports.GetFlowInstanceRequest,
) (*FlowInstanceDetail, error) {
	instance, err := s.flowRepo.GetInstance(ctx, req)
	if err != nil {
		return nil, err
	}
	events, err := s.flowRepo.ListEvents(ctx, ediports.ListFlowEventsRequest{
		TenantInfo: req.TenantInfo,
		InstanceID: instance.ID,
	})
	if err != nil {
		return nil, err
	}
	detail := &FlowInstanceDetail{Instance: instance, Events: events}
	if len(instance.State) > 0 {
		if err = sonic.Unmarshal(instance.State, &detail.Summary); err != nil {
			return nil, fmt.Errorf("read flow state: %w", err)
		}
	}
	return detail, nil
}

// ReleaseFlowInstance takes a held or escalated instance out of the exception queue once a
// person has dealt with it.
func (s *Service) ReleaseFlowInstance(
	ctx context.Context,
	req ediports.GetFlowInstanceRequest,
	actor *permissionports.RequestActor,
) (*edi.FlowInstance, error) {
	released, err := s.flows.Release(ctx, req.TenantInfo, req.ID, req.TenantInfo.UserID)
	if err != nil {
		return nil, err
	}
	s.logAction(
		released,
		actor,
		permission.OpUpdate,
		nil,
		released,
		fmt.Sprintf("EDI flow %s %s released", released.Flow, released.Key),
	)
	return released, nil
}
