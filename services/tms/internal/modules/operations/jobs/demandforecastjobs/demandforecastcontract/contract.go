package demandforecastcontract

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

const (
	FitAllWorkflowName    = "FitDemandForecastsWorkflow"
	FitTenantWorkflowName = "FitDemandForecastWorkflow"
	fitWorkflowIDPrefix   = "demand-forecast-fit-"
)

type FitPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

func (p *FitPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

func FitWorkflowID(tenantInfo pagination.TenantInfo) string {
	return fitWorkflowIDPrefix + tenantInfo.OrgID.String() + "-" + tenantInfo.BuID.String()
}

type StarterParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter
}

type Starter struct {
	workflows kernelports.WorkflowStarter
}

var _ demandforecastports.FitStarter = (*Starter)(nil)

func NewStarter(p StarterParams) demandforecastports.FitStarter {
	return &Starter{workflows: p.Workflows}
}

func (s *Starter) StartFitting(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if _, err := s.workflows.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                       FitWorkflowID(tenantInfo),
			TaskQueue:                temporaltype.TaskQueueDispatch.String(),
			WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
			StaticSummary:            "Fit the organization's freight forecast from its history",
			Priority:                 temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		FitTenantWorkflowName,
		&FitPayload{OrganizationID: tenantInfo.OrgID, BusinessUnitID: tenantInfo.BuID},
	); err != nil {
		return fmt.Errorf("start fitting the freight forecast: %w", err)
	}

	return nil
}
