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
	ToolCount         int
	RunWithoutPerson  int
	LeaveOrganization int
	OpenWithSensitive int
	Resources         []string
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
	ListSubjects(
		ctx context.Context,
		req *ListAgentSafetyRequest,
	) ([]*AgentSafetySubject, error)
	Assess(ctx context.Context, req *AssessAgentSafetyRequest) []AgentToolSafety
	ReachWarnings(req *AgentReachRequest) []AgentReachWarning
}
