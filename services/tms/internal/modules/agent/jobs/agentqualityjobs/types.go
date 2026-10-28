package agentqualityjobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualityservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	AgentQualitySweepWorkflowName          = "AgentQualitySweepWorkflow"
	AgentSuiteRunWorkflowName              = "AgentSuiteRunWorkflow"
	AgentSuiteJudgeWorkflowName            = "AgentSuiteJudgeWorkflow"
	ReconcileQualitySchedulesWorkflowName  = "ReconcileQualitySchedulesWorkflow"
	ReconcileQualitySchedulesScheduleID    = "agent-quality-schedules"
	sweepWorkflowIDPrefix                  = "agent-quality-sweep/"
	qualityScheduleOwner                   = "agent-quality"
	casesPerExecution                      = 100
	agentsPerExecution                     = 100
	reconcileQualitySchedulesOnStartWorkID = "agent-quality-schedules-on-start"
	suiteJudgeWorkflowSuffix               = "/judge"
)

type SweepPayload struct {
	OrganizationID    pulid.ID                       `json:"organizationId"`
	BusinessUnitID    pulid.ID                       `json:"businessUnitId"`
	AgentDefinitionID pulid.ID                       `json:"agentDefinitionId,omitzero"`
	Plan              *agentqualityservice.SweepPlan `json:"plan,omitempty"`
	Cursor            int                            `json:"cursor,omitempty"`
	Result            *SweepResult                   `json:"result,omitempty"`
}

func (p *SweepPayload) tenantInfo() pagination.TenantInfo {
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
	OrganizationID    pulid.ID                   `json:"organizationId"`
	BusinessUnitID    pulid.ID                   `json:"businessUnitId"`
	UserID            pulid.ID                   `json:"userId,omitzero"`
	SuiteRunID        pulid.ID                   `json:"suiteRunId"`
	AgentDefinitionID pulid.ID                   `json:"agentDefinitionId"`
	SampleSeed        int64                      `json:"sampleSeed"`
	Settings          agentquality.SuiteSettings `json:"settings"`
	DayStart          int64                      `json:"dayStart"`
	MonthStart        int64                      `json:"monthStart"`
	Cursor            int                        `json:"cursor,omitempty"`
	Replayed          int                        `json:"replayed,omitempty"`
	StopReason        string                     `json:"stopReason,omitempty"`
	Unattended        bool                       `json:"unattended,omitempty"`
	CommittedUSD      decimal.Decimal            `json:"committedUsd,omitzero"`
}

func (p *SuiteRunPayload) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type SuiteRunResult struct {
	Replayed     int                         `json:"replayed"`
	Judged       int                         `json:"judged"`
	Status       agentquality.SuiteRunStatus `json:"status"`
	QualityScore *float64                    `json:"qualityScore,omitempty"`
	Regression   bool                        `json:"regression"`
	Judging      bool                        `json:"judging,omitempty"`
	CommittedUSD decimal.Decimal             `json:"committedUsd,omitzero"`
}

type SuiteJudgePayload struct {
	Suite SuiteRunPayload                `json:"suite"`
	Batch agentqualityservice.JudgeBatch `json:"batch"`
}

func SuiteJudgeWorkflowID(suiteRunID pulid.ID) string {
	return agentquality.SuiteRunWorkflowID(suiteRunID) + suiteJudgeWorkflowSuffix
}

type ReconcileResult struct {
	Synced  int `json:"synced"`
	Removed int `json:"removed"`
	Failed  int `json:"failed"`
}
