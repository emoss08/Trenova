package factoringcontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RemittanceStageWorkflowName     = "FactoringRemittanceStageWorkflow"
	remittanceStageWorkflowIDPrefix = "factoring-remittance-stage-"
)

type RemittanceStagePayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	RemittanceID   pulid.ID `json:"remittanceId"`
	Version        int64    `json:"version"`
	GiveUp         bool     `json:"giveUp"`
}

func (p *RemittanceStagePayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

func RemittanceStageWorkflowID(remittanceID pulid.ID) string {
	return remittanceStageWorkflowIDPrefix + remittanceID.String()
}

type RemittanceStageStarterParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter `optional:"true"`
	Logger    *zap.Logger
}

type RemittanceStageStarter struct {
	workflows kernelports.WorkflowStarter
	l         *zap.Logger
}

var _ factoringports.RemittanceStageStarter = (*RemittanceStageStarter)(nil)

func NewRemittanceStageStarter(
	p RemittanceStageStarterParams,
) factoringports.RemittanceStageStarter {
	return &RemittanceStageStarter{
		workflows: p.Workflows,
		l:         p.Logger.Named("factoring-remittance-stage-starter"),
	}
}

func (s *RemittanceStageStarter) StartStage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	remittance *factoring.Remittance,
) {
	if remittance == nil {
		return
	}
	if s.workflows == nil || !s.workflows.Enabled() {
		s.l.Warn("remittance reading is not available; the PDF waits to be read again",
			zap.String("remittanceId", remittance.ID.String()))
		return
	}
	payload := &RemittanceStagePayload{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		RemittanceID:   remittance.ID,
		Version:        remittance.Version,
	}
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		_, err := s.workflows.StartWorkflow(
			committed,
			client.StartWorkflowOptions{
				ID:                       RemittanceStageWorkflowID(remittance.ID),
				TaskQueue:                temporaltype.TaskQueueBilling.String(),
				StaticSummary:            "Read factor remittance " + remittance.FileName,
				WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
				Priority:                 temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
			},
			RemittanceStageWorkflowName,
			payload,
		)
		if err != nil {
			s.l.Error("could not start reading a factor remittance",
				zap.String("remittanceId", remittance.ID.String()),
				zap.Error(err))
		}
	})
}
