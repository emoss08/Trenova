package triagewakecontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

const (
	WorkflowName             = "TriageWakeWorkflow"
	RescheduleSignal         = "triage-wake-reschedule"
	ReconcileWorkflowName    = "ReconcileTriageWakesWorkflow"
	ReconcileScheduleID      = "triage-wake-reconcile"
	RecomputeDueWorkflowName = "RecomputeTriageDueWorkflow"
)

func WorkflowIDFor(messageID pulid.ID) string { return "triage-wake:" + messageID.String() }

func RecomputeDueWorkflowIDFor(tenantInfo pagination.TenantInfo, runID pulid.ID) string {
	return "triage-recompute-due:" + tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String() +
		":" + runID.String()
}

type Payload struct {
	temporaltype.BasePayload

	MessageID pulid.ID `json:"messageId"`
}

func (p *Payload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type Schedule struct {
	Open   bool  `json:"open"`
	WakeAt int64 `json:"wakeAt"`
}

type WakeInput struct {
	Payload *Payload `json:"payload"`
	Now     int64    `json:"now"`
}

type WakeResult struct {
	Woken    bool     `json:"woken"`
	Schedule Schedule `json:"schedule"`
}

type RecomputeDuePayload struct {
	temporaltype.BasePayload

	Classifications []inboundmessage.Classification `json:"classifications"`
}

func (p *RecomputeDuePayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type Worker interface {
	WakeSchedule(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) (*Schedule, error)
	WakeIfDue(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		now int64,
	) (*WakeResult, error)
	ReconcileOverdueWakes(ctx context.Context, now int64) (int, error)
	RecomputeDueFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		classifications []inboundmessage.Classification,
	) (int64, error)
}

var _ inboundmessageports.TriageWakeScheduler = (*Scheduler)(nil)

type SchedulerParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter       `optional:"true"`
	Signals   kernelports.WorkflowSignalStarter `optional:"true"`
}

type Scheduler struct {
	workflows kernelports.WorkflowStarter
	signals   kernelports.WorkflowSignalStarter
}

func NewScheduler(p SchedulerParams) *Scheduler {
	return &Scheduler{workflows: p.Workflows, signals: p.Signals}
}

func (s *Scheduler) Enabled() bool {
	return s.workflows != nil && s.signals != nil && s.workflows.Enabled()
}

func (s *Scheduler) ScheduleWake(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) error {
	if !s.Enabled() {
		return kernelports.ErrWorkflowStarterDisabled
	}

	if _, err := s.signals.SignalWithStartWorkflow(
		ctx,
		WorkflowIDFor(messageID),
		RescheduleSignal,
		nil,
		client.StartWorkflowOptions{
			ID:                    WorkflowIDFor(messageID),
			TaskQueue:             temporaltype.TaskQueueSystem.String(),
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		},
		WorkflowName,
		&Payload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
			},
			MessageID: messageID,
		},
	); err != nil {
		return fmt.Errorf("schedule the wake-up of inbound message %s: %w", messageID, err)
	}

	return nil
}

func (s *Scheduler) CancelWake(ctx context.Context, messageID pulid.ID) error {
	if !s.Enabled() {
		return kernelports.ErrWorkflowStarterDisabled
	}

	err := s.workflows.CancelWorkflow(ctx, WorkflowIDFor(messageID), "")
	var gone *serviceerror.NotFound
	if err == nil || errors.As(err, &gone) {
		return nil
	}

	return fmt.Errorf("cancel the wake-up of inbound message %s: %w", messageID, err)
}

func (s *Scheduler) RecomputeDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	classifications []inboundmessage.Classification,
) error {
	if !s.Enabled() {
		return kernelports.ErrWorkflowStarterDisabled
	}

	if _, err := s.workflows.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        RecomputeDueWorkflowIDFor(tenantInfo, pulid.MustNew("trcd_")),
			TaskQueue: temporaltype.TaskQueueSystem.String(),
		},
		RecomputeDueWorkflowName,
		&RecomputeDuePayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
			},
			Classifications: classifications,
		},
	); err != nil {
		return fmt.Errorf("start recomputing response targets: %w", err)
	}

	return nil
}
