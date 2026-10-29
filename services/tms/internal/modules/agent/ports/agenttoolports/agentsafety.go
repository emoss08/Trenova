package agenttoolports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreach"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
	Override    *agenttooltrust.ToolRuleOverride
}

type ToolAutonomy struct {
	Answer          agenttooltrust.AutonomyAnswer
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
	Control *agentcontrol.AgentControl
}

type AgentReachWarning struct {
	Kind  agentreach.ReachWarningKind
	Tools []string
}

type ListAgentSafetyRequest struct {
	TenantInfo pagination.TenantInfo
	AgentIDs   []pulid.ID
}

type AssessAgentSafetyRequest struct {
	Subject *AgentSafetySubject
	Trust   []*agenttooltrust.ToolTrust
	Rules   map[string]*agenttooltrust.ToolRuleOverride
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
	Edges           []AgentToolSafetyEdge
	HasNextPage     bool
	TotalCount      *int
	TotalCountExact bool
}

type AgentToolPolicyEdge struct {
	View              AgentToolPolicyView
	Cursor            string
	RunsWithoutPerson *bool
}

type AgentToolPolicyPage struct {
	Edges           []AgentToolPolicyEdge
	HasNextPage     bool
	TotalCount      *int
	TotalCountExact bool
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
