package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ToolGrant struct {
	Resource  permission.Resource
	Operation permission.Operation
}

type AgentToolPolicyView struct {
	Policy      ToolPolicy
	Title       string
	Needs       *ToolGrant
	Promotable  agent.AutonomyTier
	Leaves      bool
	Explanation string
	Declared    *ToolPolicy
	Override    *agent.ToolRuleOverride
}

type ToolAutonomy struct {
	Answer          agent.AutonomyAnswer
	Tier            agent.AutonomyTier
	HeldBy          []string
	Earned          bool
	ApprovalsToNext *int
}

type AgentToolSafety struct {
	AgentID    pulid.ID
	AgentName  string
	PolicyName string
	Clean      ToolAutonomy
	Tainted    ToolAutonomy
	Policy     *AgentToolPolicyView
}

func (s *AgentToolSafety) RowID() string {
	return s.AgentID.String() + ":" + s.PolicyName
}

type AgentSafetySubject struct {
	Agent   *agentdefinition.Definition
	Control *tenant.AgentControl
}

type AgentReachWarning struct {
	Kind  agentdefinition.ReachWarningKind
	Tools []string
}

type ListAgentSafetyRequest struct {
	TenantInfo pagination.TenantInfo
	AgentIDs   []pulid.ID
}

type AssessAgentSafetyRequest struct {
	Subject *AgentSafetySubject
	Trust   []*agent.ToolTrust
	Rules   map[string]*agent.ToolRuleOverride
}

type AgentReachRequest struct {
	Subject      *AgentSafetySubject
	GrantedRoles int
}

type ListAgentToolPoliciesRequest struct {
	TenantInfo     pagination.TenantInfo
	Table          memtable.Request
	WithAttendance bool
}

type ListAgentToolSafetyRequest struct {
	TenantInfo pagination.TenantInfo
	AgentIDs   []pulid.ID
	Table      memtable.Request
}

type AgentToolSafetyEdge struct {
	Node   AgentToolSafety
	Cursor string
}

type AgentToolSafetyPage struct {
	Edges       []AgentToolSafetyEdge
	HasNextPage bool
	TotalCount  *int
}

type AgentToolPolicyEdge struct {
	View              AgentToolPolicyView
	Cursor            string
	RunsWithoutPerson *bool
}

type AgentToolPolicyPage struct {
	Edges       []AgentToolPolicyEdge
	HasNextPage bool
	TotalCount  *int
}

type AgentSafetySummary struct {
	ToolCount             int
	RunWithoutPerson      int
	LeaveOrganization     int
	OpenWithSensitive     int
	Resources             []string
	EgressCounts          []AgentEgressCount
	UnattendedTools       []string
	OpenSensitiveAgentIDs []pulid.ID
}

type RuleImpactRequest struct {
	TenantInfo pagination.TenantInfo
	ToolName   string
	Before     ToolPolicy
	After      ToolPolicy
}

type AgentToolHolders struct {
	PolicyName string
	AgentIDs   []pulid.ID
}

type AgentEgressCount struct {
	Egress agent.EgressClass
	Count  int
}

type AgentSafetyService interface {
	ToolPolicies() []AgentToolPolicyView
	ToolPolicy(name string) (AgentToolPolicyView, bool)
	ListToolPolicies(
		ctx context.Context,
		req *ListAgentToolPoliciesRequest,
	) (*AgentToolPolicyPage, error)
	ListAgentTools(
		ctx context.Context,
		req *ListAgentToolSafetyRequest,
	) (*AgentToolSafetyPage, error)
	Summary(ctx context.Context, tenantInfo pagination.TenantInfo) (*AgentSafetySummary, error)
	ToolHolders(ctx context.Context, tenantInfo pagination.TenantInfo) ([]AgentToolHolders, error)
	RuleImpact(ctx context.Context, req *RuleImpactRequest) ([]AgentToolRuleImpact, error)
	ListSubjects(
		ctx context.Context,
		req *ListAgentSafetyRequest,
	) ([]*AgentSafetySubject, error)
	Assess(ctx context.Context, req *AssessAgentSafetyRequest) []AgentToolSafety
	ReachWarnings(req *AgentReachRequest) []AgentReachWarning
}
