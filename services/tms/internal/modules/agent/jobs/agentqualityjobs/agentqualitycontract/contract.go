package agentqualitycontract

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualityengine"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

const (
	AgentQualitySweepWorkflowName = "AgentQualitySweepWorkflow"
	AgentSuiteRunWorkflowName     = "AgentSuiteRunWorkflow"
	// QualityScheduleOwner marks the schedules the quality reconciler owns.
	QualityScheduleOwner  = "agent-quality"
	sweepWorkflowIDPrefix = "agent-quality-sweep/"
)

type SweepPayload struct {
	OrganizationID    pulid.ID                      `json:"organizationId"`
	BusinessUnitID    pulid.ID                      `json:"businessUnitId"`
	AgentDefinitionID pulid.ID                      `json:"agentDefinitionId,omitzero"`
	Plan              *agentqualityengine.SweepPlan `json:"plan,omitempty"`
	Cursor            int                           `json:"cursor,omitempty"`
	Result            *SweepResult                  `json:"result,omitempty"`
}

func (p *SweepPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type SweepResult struct {
	Planned      int             `json:"planned"`
	Ran          int             `json:"ran"`
	Skipped      int             `json:"skipped"`
	Failed       int             `json:"failed"`
	CommittedUSD decimal.Decimal `json:"committedUsd,omitzero"`
}

type SuiteRunPayload struct {
	OrganizationID    pulid.ID                          `json:"organizationId"`
	BusinessUnitID    pulid.ID                          `json:"businessUnitId"`
	UserID            pulid.ID                          `json:"userId,omitzero"`
	SuiteRunID        pulid.ID                          `json:"suiteRunId"`
	AgentDefinitionID pulid.ID                          `json:"agentDefinitionId"`
	SampleSeed        int64                             `json:"sampleSeed"`
	Settings          agentqualitycontrol.SuiteSettings `json:"settings"`
	DayStart          int64                             `json:"dayStart"`
	MonthStart        int64                             `json:"monthStart"`
	Cursor            int                               `json:"cursor,omitempty"`
	Replayed          int                               `json:"replayed,omitempty"`
	StopReason        string                            `json:"stopReason,omitempty"`
	Unattended        bool                              `json:"unattended,omitempty"`
	CommittedUSD      decimal.Decimal                   `json:"committedUsd,omitzero"`
}

func (p *SuiteRunPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type ReconcileResult struct {
	Synced  int `json:"synced"`
	Removed int `json:"removed"`
	Failed  int `json:"failed"`
}

func SweepWorkflowID(orgID pulid.ID) string {
	return sweepWorkflowIDPrefix + orgID.String()
}

func EvaluationPriority(orgID pulid.ID) temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflowcontract.PriorityEvaluation,
		FairnessKey: orgID.String(),
	}
}
