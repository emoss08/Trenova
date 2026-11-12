package triagewakecontract

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

const (
	WakeDueWorkflowName      = "WakeDueTriageWorkflow"
	WakeDueScheduleID        = "triage-wake-due"
	RecomputeDueWorkflowName = "RecomputeTriageDueWorkflow"
)

func RecomputeDueWorkflowIDFor(tenantInfo pagination.TenantInfo, runID pulid.ID) string {
	return "triage-recompute-due:" + tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String() +
		":" + runID.String()
}

type RecomputeDuePayload struct {
	temporaltype.BasePayload

	Classifications []inboundmessage.Classification `json:"classifications"`
}

func (p *RecomputeDuePayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

// Worker is what the triage jobs call: the minute sweep that wakes every
// parked message whose time has come, and the recompute of response targets.
type Worker interface {
	WakeDue(ctx context.Context, now int64) (int, error)
	RecomputeDueFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		classifications []inboundmessage.Classification,
	) (int64, error)
}

var _ inboundmessageports.TriageDueRecomputer = (*DueRecomputer)(nil)

type DueRecomputerParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter
}

// DueRecomputer starts the recompute of a tenant's response targets after it
// changes them.
type DueRecomputer struct {
	workflows kernelports.WorkflowStarter
}

func NewDueRecomputer(p DueRecomputerParams) *DueRecomputer {
	return &DueRecomputer{workflows: p.Workflows}
}

func (s *DueRecomputer) RecomputeDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	classifications []inboundmessage.Classification,
) error {
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
