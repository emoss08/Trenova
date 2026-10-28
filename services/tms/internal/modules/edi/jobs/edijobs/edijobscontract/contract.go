// Package edijobscontract is how the API hands EDI flow work to a worker: the timer that wakes
// a flow instance when it is due, and the shipment events a partner's flow follows.
package edijobscontract

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	FlowEventWorkflowName = "EDIFlowShipmentEventWorkflow"
	// RoundsPerRun bounds a timer workflow's history: it continues as new after this many
	// wakes.
	RoundsPerRun = 50
)

// FlowTimerWorkflowID is one per flow instance, so scheduling again moves the one timer.
func FlowTimerWorkflowID(instanceID pulid.ID) string {
	return "edi-flow-timer:" + instanceID.String()
}

// FlowEventWorkflowID is one per shipment event, so an event is given to the flows once.
func FlowEventWorkflowID(eventID pulid.ID) string {
	return "edi-flow-event:" + eventID.String()
}

// FlowTimerInput starts a timer workflow; its wake signal carries the instant, in Unix
// seconds, the instance is next due.
type FlowTimerInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	InstanceID pulid.ID              `json:"instanceId"`
	WakeAt     int64                 `json:"wakeAt"`
}

type FlowEventInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	EventID    pulid.ID              `json:"eventId"`
}

type TimerSchedulerParams struct {
	fx.In

	Logger  *zap.Logger
	Signals kernelports.WorkflowSignalStarter `optional:"true"`
}

type TimerScheduler struct {
	l       *zap.Logger
	signals kernelports.WorkflowSignalStarter
}

func NewTimerScheduler(p TimerSchedulerParams) ladingports.FlowTimerScheduler {
	return &TimerScheduler{l: p.Logger.Named("job.edi-flow-timer"), signals: p.Signals}
}

// Schedule starts the instance's timer workflow, or moves the one already waiting.
func (s *TimerScheduler) Schedule(
	ctx context.Context,
	req *ladingports.ScheduleFlowTimerRequest,
) error {
	if s.signals == nil {
		return nil
	}
	tenant := pagination.TenantInfo{OrgID: req.TenantInfo.OrgID, BuID: req.TenantInfo.BuID}
	_, err := s.signals.SignalWithStartWorkflow(
		ctx,
		FlowTimerWorkflowID(req.InstanceID),
		temporaltype.EDIFlowTimerWakeSignal,
		req.WakeAt,
		client.StartWorkflowOptions{
			ID:            FlowTimerWorkflowID(req.InstanceID),
			TaskQueue:     temporaltype.EDITaskQueue,
			StaticSummary: "Wake an EDI flow when it is due",
		},
		temporaltype.EDIFlowTimerWorkflowName,
		&FlowTimerInput{TenantInfo: tenant, InstanceID: req.InstanceID, WakeAt: req.WakeAt},
	)
	return err
}

type ShipmentObserverParams struct {
	fx.In

	Logger    *zap.Logger
	Flows     ladingports.FlowRepository
	Workflows kernelports.WorkflowStarter `optional:"true"`
}

type ShipmentObserver struct {
	l         *zap.Logger
	flows     ladingports.FlowRepository
	workflows kernelports.WorkflowStarter
}

func NewShipmentObserver(p ShipmentObserverParams) shipmenteventports.ShipmentEventObserver {
	return &ShipmentObserver{
		l:         p.Logger.Named("job.edi-flow-event"),
		flows:     p.Flows,
		workflows: p.Workflows,
	}
}

// OnShipmentEvent hands an event to a worker when a partner's flow follows the shipment; the
// check is one indexed read, and a shipment no flow follows costs nothing more.
func (o *ShipmentObserver) OnShipmentEvent(ctx context.Context, event *shipmentevent.Event) error {
	if o.workflows == nil || !o.workflows.Enabled() || !ediservice.FlowShipmentEvent(event.Type) {
		return nil
	}
	tenant := pagination.TenantInfo{OrgID: event.OrganizationID, BuID: event.BusinessUnitID}
	instances, err := o.flows.ListOpenInstances(
		dbscope.WithTenant(ctx, tenant.DBTenant()),
		&ladingports.ListShipmentFlowInstancesRequest{
			TenantInfo: tenant,
			ShipmentID: event.ShipmentID,
		},
	)
	if err != nil || len(instances) == 0 {
		return err
	}
	_, err = o.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    FlowEventWorkflowID(event.ID),
		TaskQueue:             temporaltype.EDITaskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		StaticSummary:         "Give a shipment event to the partners' EDI flows",
	}, FlowEventWorkflowName, &FlowEventInput{TenantInfo: tenant, EventID: event.ID})
	var started *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &started) {
		return nil
	}
	return err
}
