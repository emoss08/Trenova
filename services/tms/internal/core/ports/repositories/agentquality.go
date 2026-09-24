package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrSuiteRunAlreadyRunning = errors.New("this agent's suite is already running")
	ErrSuiteRunSweepKeyTaken  = errors.New("this sweep already opened a run for the agent")
)

type GetAgentSuiteRunRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetAgentSuiteRunBySweepKeyRequest struct {
	SweepKey   string
	TenantInfo pagination.TenantInfo
}

type LastAgentSuiteRunRequest struct {
	AgentDefinitionID pulid.ID
	TenantInfo        pagination.TenantInfo
	Statuses          []agentquality.SuiteRunStatus
	Before            int64
	ExcludeID         pulid.ID
}

type ListAgentSuiteRunsRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Statuses          []agentquality.SuiteRunStatus
	Limit             int
	Offset            int
	IncludeTotalCount bool
}

type AgentSuiteRunPage struct {
	Items       []*agentquality.SuiteRun
	HasNextPage bool
	TotalCount  *int
}

type AgentSuiteRunHistoryRequest struct {
	TenantInfo         pagination.TenantInfo
	AgentDefinitionIDs []pulid.ID
	Since              int64
	PerAgent           int
}

type LatestAgentSuiteRunsRequest struct {
	TenantInfo         pagination.TenantInfo
	AgentDefinitionIDs []pulid.ID
	Statuses           []agentquality.SuiteRunStatus
	Limit              int
}

type AgentSuiteRunOverviewRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

type AgentSuiteRunOverview struct {
	Runs        int `bun:"runs"`
	Regressions int `bun:"regressions"`
}

type ListAgentQualityAgentsRequest struct {
	TenantInfo        pagination.TenantInfo
	Limit             int
	Offset            int
	IncludeTotalCount bool
}

type AgentQualityAgentRow struct {
	ID      pulid.ID `bun:"id"`
	Name    string   `bun:"name"`
	Enabled bool     `bun:"enabled"`
}

type AgentQualityAgentRows struct {
	Items       []*AgentQualityAgentRow
	HasNextPage bool
	TotalCount  *int
}

type AgentSuiteRunRepository interface {
	Create(ctx context.Context, entity *agentquality.SuiteRun) (*agentquality.SuiteRun, error)
	Update(ctx context.Context, entity *agentquality.SuiteRun) (*agentquality.SuiteRun, error)
	GetByID(ctx context.Context, req GetAgentSuiteRunRequest) (*agentquality.SuiteRun, error)
	GetBySweepKey(
		ctx context.Context,
		req GetAgentSuiteRunBySweepKeyRequest,
	) (*agentquality.SuiteRun, error)
	Last(ctx context.Context, req LastAgentSuiteRunRequest) (*agentquality.SuiteRun, error)
	List(ctx context.Context, req ListAgentSuiteRunsRequest) (*AgentSuiteRunPage, error)
	History(
		ctx context.Context,
		req AgentSuiteRunHistoryRequest,
	) ([]*agentquality.SuiteRun, error)
	Latest(
		ctx context.Context,
		req LatestAgentSuiteRunsRequest,
	) ([]*agentquality.SuiteRun, error)
	Overview(ctx context.Context, req AgentSuiteRunOverviewRequest) (*AgentSuiteRunOverview, error)
	ListAgents(
		ctx context.Context,
		req ListAgentQualityAgentsRequest,
	) (*AgentQualityAgentRows, error)
}

type QualityScheduleTarget struct {
	OrganizationID       pulid.ID              `bun:"organization_id"`
	BusinessUnitID       pulid.ID              `bun:"business_unit_id"`
	OrganizationName     string                `bun:"organization_name"`
	OrganizationTimezone string                `bun:"organization_timezone"`
	HasActiveCases       bool                  `bun:"has_active_cases"`
	Control              *agentquality.Control `bun:"-"`
}

func (t *QualityScheduleTarget) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: t.OrganizationID, BuID: t.BusinessUnitID}
}

type ListQualityScheduleTargetsRequest struct {
	AfterOrganizationID pulid.ID
	Limit               int
}

type AgentQualityControlRepository interface {
	Get(ctx context.Context, tenant pagination.TenantInfo) (*agentquality.Control, error)
	Upsert(ctx context.Context, entity *agentquality.Control) (*agentquality.Control, error)
	ScheduleTarget(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*QualityScheduleTarget, error)
	ListScheduleTargets(
		ctx context.Context,
		req ListQualityScheduleTargetsRequest,
	) ([]*QualityScheduleTarget, error)
}
