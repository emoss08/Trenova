package agentqualityports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrSuiteRunAlreadyRunning = errors.New("this agent's suite is already running")

var ErrSuiteRunSweepKeyTaken = errors.New("this sweep already opened a run for the agent")

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
	Statuses          []agentqualitysuite.SuiteRunStatus
	Before            int64
	ExcludeID         pulid.ID
}

type ListAgentSuiteRunConnectionRequest struct {
	Filter            *pagination.QueryOptions `json:"filter"`
	Cursor            pagination.CursorInfo    `json:"-"`
	AgentDefinitionID pulid.ID                 `json:"agentDefinitionId"`
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
	Statuses           []agentqualitysuite.SuiteRunStatus
	Limit              int
}

type AgentSuiteRunOverviewRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

type AgentSuiteRunOverview struct {
	Runs        int
	Regressions int
}

type ListAgentQualityAgentsQuery struct {
	TenantInfo        pagination.TenantInfo
	Limit             int
	Offset            int
	IncludeTotalCount bool
}

type AgentQualityAgentRow struct {
	ID      pulid.ID
	Name    string
	Enabled bool
}

type AgentQualityAgentRows struct {
	Items           []*AgentQualityAgentRow
	HasNextPage     bool
	TotalCount      *int
	TotalCountExact bool
}

type AgentSuiteRunRepository interface {
	Create(
		ctx context.Context,
		entity *agentqualitysuite.SuiteRun,
	) (*agentqualitysuite.SuiteRun, error)
	Update(
		ctx context.Context,
		entity *agentqualitysuite.SuiteRun,
	) (*agentqualitysuite.SuiteRun, error)
	GetByID(ctx context.Context, req GetAgentSuiteRunRequest) (*agentqualitysuite.SuiteRun, error)
	GetBySweepKey(
		ctx context.Context,
		req GetAgentSuiteRunBySweepKeyRequest,
	) (*agentqualitysuite.SuiteRun, error)
	Last(ctx context.Context, req LastAgentSuiteRunRequest) (*agentqualitysuite.SuiteRun, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentSuiteRunConnectionRequest,
	) (*pagination.CursorListResult[*agentqualitysuite.SuiteRun], error)
	History(
		ctx context.Context,
		req AgentSuiteRunHistoryRequest,
	) ([]*agentqualitysuite.SuiteRun, error)
	Latest(
		ctx context.Context,
		req LatestAgentSuiteRunsRequest,
	) ([]*agentqualitysuite.SuiteRun, error)
	Overview(ctx context.Context, req AgentSuiteRunOverviewRequest) (*AgentSuiteRunOverview, error)
	ListAgents(
		ctx context.Context,
		req ListAgentQualityAgentsQuery,
	) (*AgentQualityAgentRows, error)
}

type QualityScheduleTarget struct {
	OrganizationID       pulid.ID
	BusinessUnitID       pulid.ID
	OrganizationName     string
	OrganizationTimezone string
	HasActiveCases       bool
	Control              *agentqualitycontrol.Control
}

func (t *QualityScheduleTarget) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: t.OrganizationID, BuID: t.BusinessUnitID}
}

type ListQualityScheduleTargetsRequest struct {
	AfterOrganizationID pulid.ID
	Limit               int
}

type AgentQualityControlRepository interface {
	Get(ctx context.Context, tenant pagination.TenantInfo) (*agentqualitycontrol.Control, error)
	Upsert(
		ctx context.Context,
		entity *agentqualitycontrol.Control,
	) (*agentqualitycontrol.Control, error)
	ScheduleTarget(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*QualityScheduleTarget, error)
	ListScheduleTargets(
		ctx context.Context,
		req ListQualityScheduleTargetsRequest,
	) ([]*QualityScheduleTarget, error)
}
