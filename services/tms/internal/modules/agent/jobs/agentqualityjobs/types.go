package agentqualityjobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualityengine"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentqualityjobs/agentqualitycontract"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	AgentSuiteJudgeWorkflowName            = "AgentSuiteJudgeWorkflow"
	ReconcileQualitySchedulesWorkflowName  = "ReconcileQualitySchedulesWorkflow"
	ReconcileQualitySchedulesScheduleID    = "agent-quality-schedules"
	casesPerExecution                      = 100
	agentsPerExecution                     = 100
	reconcileQualitySchedulesOnStartWorkID = "agent-quality-schedules-on-start"
	suiteJudgeWorkflowSuffix               = "/judge"
)

type SuiteRunResult struct {
	Replayed     int                              `json:"replayed"`
	Judged       int                              `json:"judged"`
	Status       agentqualitysuite.SuiteRunStatus `json:"status"`
	QualityScore *float64                         `json:"qualityScore,omitempty"`
	Regression   bool                             `json:"regression"`
	Judging      bool                             `json:"judging,omitempty"`
	CommittedUSD decimal.Decimal                  `json:"committedUsd,omitzero"`
}

type SuiteJudgePayload struct {
	Suite agentqualitycontract.SuiteRunPayload `json:"suite"`
	Batch agentqualityengine.JudgeBatch        `json:"batch"`
}

func SuiteJudgeWorkflowID(suiteRunID pulid.ID) string {
	return agentqualitycontrol.SuiteRunWorkflowID(suiteRunID) + suiteJudgeWorkflowSuffix
}
