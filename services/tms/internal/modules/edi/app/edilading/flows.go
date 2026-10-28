package edilading

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxStepAttempts = 4
	maxFlowKeyLen   = 300
)

var errNoFlowKey = errors.New("edi: the flow names no key for this event")

type FlowsParams struct {
	fx.In

	Logger  *zap.Logger
	Runtime *Runtime
	Flows   ladingports.FlowRepository
	Timers  ladingports.FlowTimerScheduler `optional:"true"`
}

type Flows struct {
	l       *zap.Logger
	runtime *Runtime
	flows   ladingports.FlowRepository
	timers  ladingports.FlowTimerScheduler
}

func NewFlows(p FlowsParams) *Flows {
	return &Flows{
		l:       p.Logger.Named("edi.lading.flows"),
		runtime: p.Runtime,
		flows:   p.Flows,
		timers:  p.Timers,
	}
}

type DispatchRequest struct {
	TenantInfo  pagination.TenantInfo
	PartnerID   pulid.ID
	Program     *Program
	Flow        string
	Key         string
	FallbackKey string
	Event       lading.FlowEvent
	ShipmentID  pulid.ID
	TransferID  pulid.ID
	MessageID   pulid.ID
	ActorID     pulid.ID
	Now         time.Time
}

type Dispatched struct {
	Program  *Program
	Instance *edi.FlowInstance
	Actions  []lading.FlowAction
}

// Dispatch feeds one event to the flow instance it belongs to, opening the instance on its
// first event, and returns the actions Trenova takes. The caller carries them out.
func (f *Flows) Dispatch(ctx context.Context, req *DispatchRequest) (*Dispatched, error) {
	if req.Program == nil {
		program, err := f.runtime.Active(ctx, req.TenantInfo, req.PartnerID)
		if err != nil {
			return nil, err
		}
		req.Program = program
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	settings, err := Settings(req.Program.Project)
	if err != nil {
		return nil, err
	}
	key, err := f.key(ctx, req, settings)
	if err != nil {
		return nil, err
	}
	instance, err := f.flows.OpenInstance(ctx, &edi.FlowInstance{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		EDIPartnerID:   req.PartnerID,
		ProjectID:      req.Program.Project.ID,
		VersionID:      req.Program.Version.ID,
		Flow:           req.Flow,
		Key:            key,
		Status:         edi.FlowInstanceStatusOpen,
		ShipmentID:     req.ShipmentID,
		TransferID:     req.TransferID,
		LastEventAt:    req.Now.Unix(),
	})
	if err != nil {
		return nil, err
	}
	return f.step(ctx, req, instance, settings)
}

// DispatchTo feeds an event to an instance already open.
func (f *Flows) DispatchTo(
	ctx context.Context,
	instance *edi.FlowInstance,
	req *DispatchRequest,
) (*Dispatched, error) {
	if req.Program == nil {
		program, err := f.runtime.Active(ctx, req.TenantInfo, instance.EDIPartnerID)
		if err != nil {
			return nil, err
		}
		req.Program = program
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	req.PartnerID = instance.EDIPartnerID
	req.Flow = instance.Flow
	req.Key = instance.Key
	settings, err := Settings(req.Program.Project)
	if err != nil {
		return nil, err
	}
	return f.step(ctx, req, instance, settings)
}

// Attach records the transfer or shipment an instance's load became, so later events and a
// planner's decision find it.
func (f *Flows) Attach(
	ctx context.Context,
	instance *edi.FlowInstance,
	transferID, shipmentID pulid.ID,
) (*edi.FlowInstance, error) {
	if transferID.IsNotNil() {
		instance.TransferID = transferID
	}
	if shipmentID.IsNotNil() {
		instance.ShipmentID = shipmentID
	}
	return f.flows.SaveInstance(ctx, instance)
}

func (f *Flows) key(
	ctx context.Context,
	req *DispatchRequest,
	settings map[string]jsonutils.RawJSON,
) (string, error) {
	key := req.Key
	if key == "" {
		found, err := f.runtime.client.FlowKey(ctx, req.Program.Encoded, &lading.FlowRequest{
			Flow:     req.Flow,
			Event:    req.Event,
			Now:      instant(req.Now),
			Settings: settings,
		})
		if err != nil {
			return "", err
		}
		key = found
	}
	if key == "" {
		key = req.FallbackKey
	}
	if key == "" {
		return "", errNoFlowKey
	}
	if len(key) > maxFlowKeyLen {
		key = key[:maxFlowKeyLen]
	}
	return key, nil
}

func (f *Flows) step(
	ctx context.Context,
	req *DispatchRequest,
	instance *edi.FlowInstance,
	settings map[string]jsonutils.RawJSON,
) (*Dispatched, error) {
	for attempt := 1; ; attempt++ {
		step, err := f.runtime.client.FlowStep(ctx, req.Program.Encoded, &lading.FlowRequest{
			Flow:     req.Flow,
			State:    instance.State,
			Event:    req.Event,
			Now:      instant(req.Now),
			Settings: settings,
		})
		if err != nil {
			return nil, err
		}
		if step.Error != nil {
			f.journal(ctx, instance, req, nil, step.Error)
			return nil, step.Error.BusinessError()
		}
		previousWake := instance.WakeAt
		apply(instance, req, step)
		saved, saveErr := f.flows.SaveInstance(ctx, instance)
		if saveErr == nil {
			f.journal(ctx, saved, req, step.Actions, nil)
			f.schedule(ctx, req.TenantInfo, saved, previousWake)
			return &Dispatched{Program: req.Program, Instance: saved, Actions: step.Actions}, nil
		}
		if !dberror.IsVersionConflict(saveErr) && !errortypes.IsVersionMismatchError(saveErr) ||
			attempt == maxStepAttempts {
			return nil, saveErr
		}
		instance, err = f.flows.GetInstance(ctx, ladingports.GetFlowInstanceRequest{
			TenantInfo: req.TenantInfo,
			ID:         instance.ID,
		})
		if err != nil {
			return nil, err
		}
	}
}

func apply(instance *edi.FlowInstance, req *DispatchRequest, step *lading.FlowStep) {
	now := req.Now.Unix()
	instance.State = step.State
	instance.VersionID = req.Program.Version.ID
	instance.ProjectID = req.Program.Project.ID
	instance.WakeAt = step.Wake
	instance.LastEventAt = now
	if req.ShipmentID.IsNotNil() {
		instance.ShipmentID = req.ShipmentID
	}
	if req.TransferID.IsNotNil() {
		instance.TransferID = req.TransferID
	}
	for index := range step.Actions {
		action := &step.Actions[index]
		switch action.Kind {
		case lading.ActionHold:
			instance.Status = edi.FlowInstanceStatusHeld
			instance.Reason = action.Message
			instance.HeldAt = &now
			instance.ResolvedAt = nil
			instance.ResolvedByID = ""
		case lading.ActionEscalate:
			instance.Status = edi.FlowInstanceStatusEscalated
			instance.Reason = action.Message
			instance.HeldAt = &now
			instance.ResolvedAt = nil
			instance.ResolvedByID = ""
		}
	}
	var summary lading.FlowSummary
	if err := sonic.Unmarshal(step.State, &summary); err == nil &&
		instance.Status == edi.FlowInstanceStatusOpen && finished(&summary) && step.Wake == nil {
		instance.Status = edi.FlowInstanceStatusClosed
	}
}

func finished(summary *lading.FlowSummary) bool {
	switch summary.Load.State {
	case lading.LoadCancelled, lading.LoadInvoiced:
		return summary.AwaitingResponse == nil && len(summary.Pending) == 0
	default:
		return false
	}
}

func (f *Flows) journal(
	ctx context.Context,
	instance *edi.FlowInstance,
	req *DispatchRequest,
	actions []lading.FlowAction,
	failure *wasmmodule.Message,
) {
	detail, err := sonic.Marshal(eventDetail(&req.Event))
	if err != nil {
		detail = []byte("{}")
	}
	if actions == nil {
		actions = []lading.FlowAction{}
	}
	stripped := make([]lading.FlowAction, len(actions))
	for index := range actions {
		stripped[index] = actions[index]
		stripped[index].Document = nil
	}
	if err = f.flows.AppendEvent(ctx, &edi.FlowEvent{
		OrganizationID: instance.OrganizationID,
		BusinessUnitID: instance.BusinessUnitID,
		InstanceID:     instance.ID,
		Kind:           req.Event.Kind,
		Detail:         detail,
		Actions:        stripped,
		Error:          failure,
		MessageID:      req.MessageID,
		ActorID:        req.ActorID,
	}); err != nil {
		f.l.Warn("failed to journal a flow event",
			zap.String("instanceId", instance.ID.String()),
			zap.Error(err),
		)
	}
}

type flowEventDetail struct {
	Set      string                `json:"set,omitempty"`
	Event    string                `json:"event,omitempty"`
	Decision string                `json:"decision,omitempty"`
	Reason   string                `json:"reason,omitempty"`
	ID       string                `json:"id,omitempty"`
	Control  string                `json:"control,omitempty"`
	Accepted *bool                 `json:"accepted,omitempty"`
	Outcomes []lading.OutcomeInput `json:"outcomes,omitempty"`
	Load     *lading.LoadState     `json:"load,omitempty"`
}

func eventDetail(event *lading.FlowEvent) flowEventDetail {
	detail := flowEventDetail{
		Set:      event.Set,
		Event:    event.Event,
		Decision: event.Decision,
		Reason:   event.Reason,
		ID:       event.ID,
		Control:  event.Control,
		Outcomes: event.Outcomes,
		Load:     event.Load,
	}
	if event.Kind == lading.EventAck {
		accepted := event.Accepted
		detail.Accepted = &accepted
	}
	return detail
}

func (f *Flows) schedule(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	instance *edi.FlowInstance,
	previous *int64,
) {
	if f.timers == nil || instance.WakeAt == nil {
		return
	}
	if previous != nil && *previous == *instance.WakeAt {
		return
	}
	if err := f.timers.Schedule(ctx, &ladingports.ScheduleFlowTimerRequest{
		TenantInfo: tenantInfo,
		InstanceID: instance.ID,
		WakeAt:     *instance.WakeAt,
	}); err != nil {
		f.l.Error("failed to schedule a flow timer",
			zap.String("instanceId", instance.ID.String()),
			zap.Int64("wakeAt", *instance.WakeAt),
			zap.Error(err),
		)
	}
}

// Release takes an instance out of the exception queue after a person dealt with it.
func (f *Flows) Release(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	instanceID, actorID pulid.ID,
) (*edi.FlowInstance, error) {
	instance, err := f.flows.GetInstance(ctx, ladingports.GetFlowInstanceRequest{
		TenantInfo: tenantInfo,
		ID:         instanceID,
	})
	if err != nil {
		return nil, err
	}
	if instance.Status != edi.FlowInstanceStatusHeld &&
		instance.Status != edi.FlowInstanceStatusEscalated {
		return nil, errortypes.NewBusinessError(
			"This flow instance is {0}, not held or escalated",
			string(instance.Status),
		)
	}
	now := time.Now().Unix()
	instance.Status = edi.FlowInstanceStatusOpen
	instance.ResolvedAt = &now
	instance.ResolvedByID = actorID
	saved, err := f.flows.SaveInstance(ctx, instance)
	if err != nil {
		return nil, err
	}
	detail, _ := sonic.Marshal(map[string]string{"status": string(edi.FlowInstanceStatusOpen)})
	if err = f.flows.AppendEvent(ctx, &edi.FlowEvent{
		OrganizationID: saved.OrganizationID,
		BusinessUnitID: saved.BusinessUnitID,
		InstanceID:     saved.ID,
		Kind:           "released",
		Detail:         detail,
		ActorID:        actorID,
	}); err != nil {
		return nil, fmt.Errorf("journal the release: %w", err)
	}
	return saved, nil
}
