package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultAgentQualityWindowDays = 30
	MaxAgentQualityWindowDays     = 365
	DefaultAgentQualityPageSize   = 25
	MaxAgentQualityPageSize       = 100
	AgentQualityHistoryPoints     = 30
)

type AgentQualityPoint struct {
	SuiteRunID   pulid.ID                    `json:"suiteRunId"`
	At           int64                       `json:"at"`
	QualityScore float64                     `json:"qualityScore"`
	Status       agentquality.SuiteRunStatus `json:"status"`
	Regression   bool                        `json:"regression"`
}

type AgentQualityOverviewRequest struct {
	TenantInfo     pagination.TenantInfo
	WindowDays     int
	IncludeRatings bool
}

type AgentQualityOverview struct {
	WindowDays          int      `json:"windowDays"`
	Since               int64    `json:"since"`
	RatingsVisible      bool     `json:"ratingsVisible"`
	Satisfaction        *float64 `json:"satisfaction"`
	Ratings             int      `json:"ratings"`
	QualityScore        *float64 `json:"qualityScore"`
	AgentsScored        int      `json:"agentsScored"`
	SuiteRuns           int      `json:"suiteRuns"`
	Regressions         int      `json:"regressions"`
	OpenRegressions     int      `json:"openRegressions"`
	EvalSpendMonthUSD   string   `json:"evalSpendMonthUsd"`
	EvalUnpricedCalls   int      `json:"evalUnpricedCalls"`
	MonthlyBudgetUSD    string   `json:"monthlyBudgetUsd"`
	MonthStartedAt      int64    `json:"monthStartedAt"`
	SweepEnabled        bool     `json:"sweepEnabled"`
	NextSweepHourLocal  int      `json:"nextSweepHourLocal"`
	NextSweepTimezone   string   `json:"nextSweepTimezone"`
	AgentsWithCases     int      `json:"agentsWithCases"`
	JudgeEnabled        bool     `json:"judgeEnabled"`
	RegressionThreshold float64  `json:"regressionThreshold"`
}

type AgentQualityRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	WindowDays        int
	IncludeRatings    bool
	ViewerID          pulid.ID
}

type AgentWorstRatedAnswer struct {
	TargetType        aifeedback.TargetType `json:"targetType"`
	TargetID          pulid.ID              `json:"targetId"`
	TargetPart        string                `json:"targetPart"`
	Positive          int                   `json:"positive"`
	Negative          int                   `json:"negative"`
	LastRatedAt       int64                 `json:"lastRatedAt"`
	Sample            *aifeedback.Feedback  `json:"sample"`
	ThreadID          *pulid.ID             `json:"threadId"`
	CanOpenThread     bool                  `json:"canOpenThread"`
	AgentDefinitionID *pulid.ID             `json:"agentDefinitionId"`
	AgentName         string                `json:"agentName"`
}

type AgentQualityDetail struct {
	AgentDefinitionID  pulid.ID                 `json:"agentDefinitionId"`
	AgentName          string                   `json:"agentName"`
	Enabled            bool                     `json:"enabled"`
	WindowDays         int                      `json:"windowDays"`
	Since              int64                    `json:"since"`
	RatingsVisible     bool                     `json:"ratingsVisible"`
	Satisfaction       *float64                 `json:"satisfaction"`
	Ratings            int                      `json:"ratings"`
	SatisfactionPoints []*AgentFeedbackDay      `json:"satisfactionPoints"`
	QualityPoints      []*AgentQualityPoint     `json:"qualityPoints"`
	WorstRated         []*AgentWorstRatedAnswer `json:"worstRated"`
	LastSuiteRun       *agentquality.SuiteRun   `json:"lastSuiteRun"`
	ActiveCases        int                      `json:"activeCases"`
}

type ListAgentQualityAgentsRequest struct {
	TenantInfo        pagination.TenantInfo
	WindowDays        int
	IncludeRatings    bool
	First             int
	After             string
	IncludeTotalCount bool
}

type AgentQualityAgent struct {
	AgentDefinitionID pulid.ID               `json:"agentDefinitionId"`
	Name              string                 `json:"name"`
	Enabled           bool                   `json:"enabled"`
	Satisfaction      *float64               `json:"satisfaction"`
	SatisfactionDelta *float64               `json:"satisfactionDelta"`
	Ratings           int                    `json:"ratings"`
	QualityScore      *float64               `json:"qualityScore"`
	QualityPoints     []*AgentQualityPoint   `json:"qualityPoints"`
	LastSuiteRun      *agentquality.SuiteRun `json:"lastSuiteRun"`
	OpenRegression    bool                   `json:"openRegression"`
	RatingsVisible    bool                   `json:"ratingsVisible"`
}

type AgentQualityAgentEdge struct {
	Node   *AgentQualityAgent `json:"node"`
	Cursor string             `json:"cursor"`
}

type AgentQualityAgentPage struct {
	Edges       []*AgentQualityAgentEdge `json:"edges"`
	HasNextPage bool                     `json:"hasNextPage"`
	TotalCount  *int                     `json:"totalCount"`
}

type ListAgentWorstRatedRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	WindowDays        int
	First             int
	After             string
	ViewerID          pulid.ID
}

type AgentWorstRatedEdge struct {
	Node   *AgentWorstRatedAnswer `json:"node"`
	Cursor string                 `json:"cursor"`
}

type AgentWorstRatedPage struct {
	Edges       []*AgentWorstRatedEdge `json:"edges"`
	HasNextPage bool                   `json:"hasNextPage"`
}

type ListAgentSuiteRunsInput struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Statuses          []agentquality.SuiteRunStatus
	First             int
	After             string
	IncludeTotalCount bool
}

type AgentSuiteRunEdge struct {
	Node   *agentquality.SuiteRun `json:"node"`
	Cursor string                 `json:"cursor"`
}

type AgentSuiteRunConnection struct {
	Edges       []*AgentSuiteRunEdge `json:"edges"`
	HasNextPage bool                 `json:"hasNextPage"`
	TotalCount  *int                 `json:"totalCount"`
}

type ListAgentSuiteRunCasesRequest struct {
	TenantInfo        pagination.TenantInfo
	SuiteRunID        pulid.ID
	First             int
	After             string
	IncludeTotalCount bool
}

type AgentSuiteRunCaseEdge struct {
	Node   *agent.Evaluation `json:"node"`
	Cursor string            `json:"cursor"`
}

type AgentSuiteRunCasePage struct {
	Edges       []*AgentSuiteRunCaseEdge `json:"edges"`
	HasNextPage bool                     `json:"hasNextPage"`
	TotalCount  *int                     `json:"totalCount"`
}

type RunAgentSuiteRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
}

type AgentQualityService interface {
	GetControl(ctx context.Context, tenant pagination.TenantInfo) (*agentquality.Control, error)
	UpdateControl(
		ctx context.Context,
		entity *agentquality.Control,
		actor *RequestActor,
	) (*agentquality.Control, error)
	Overview(ctx context.Context, req *AgentQualityOverviewRequest) (*AgentQualityOverview, error)
	AgentQuality(ctx context.Context, req *AgentQualityRequest) (*AgentQualityDetail, error)
	ListAgents(
		ctx context.Context,
		req *ListAgentQualityAgentsRequest,
	) (*AgentQualityAgentPage, error)
	ListWorstRated(
		ctx context.Context,
		req *ListAgentWorstRatedRequest,
	) (*AgentWorstRatedPage, error)
	ListSuiteRuns(
		ctx context.Context,
		req *ListAgentSuiteRunsInput,
	) (*AgentSuiteRunConnection, error)
	GetSuiteRun(
		ctx context.Context,
		id pulid.ID,
		tenant pagination.TenantInfo,
	) (*agentquality.SuiteRun, error)
	ListSuiteRunCases(
		ctx context.Context,
		req *ListAgentSuiteRunCasesRequest,
	) (*AgentSuiteRunCasePage, error)
	RunSuite(
		ctx context.Context,
		req *RunAgentSuiteRequest,
		actor *RequestActor,
	) (*agentquality.SuiteRun, error)
}

type AgentQualityScheduler interface {
	Sync(ctx context.Context, tenant pagination.TenantInfo)
}

type AgentSuiteRunStart struct {
	TenantInfo        pagination.TenantInfo
	UserID            pulid.ID
	SuiteRunID        pulid.ID
	AgentDefinitionID pulid.ID
	SampleSeed        int64
	Settings          agentquality.SuiteSettings
	DayStart          int64
	MonthStart        int64
}

type AgentSuiteStarter interface {
	StartSuiteRun(ctx context.Context, start *AgentSuiteRunStart) (string, error)
}
