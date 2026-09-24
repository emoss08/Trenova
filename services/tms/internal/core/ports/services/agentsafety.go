package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
	PolicyName string
	Clean      ToolAutonomy
	Tainted    ToolAutonomy
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
	TenantInfo        pagination.TenantInfo
	First             int
	After             string
	Query             string
	Egress            agent.EgressClass
	Resource          string
	Kind              agent.ToolKind
	RunsWithoutPerson *bool
	IncludeTotalCount bool
}

type AgentToolPolicyEdge struct {
	View   AgentToolPolicyView
	Cursor string
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
	Summary(ctx context.Context, tenantInfo pagination.TenantInfo) (*AgentSafetySummary, error)
	ListSubjects(
		ctx context.Context,
		req *ListAgentSafetyRequest,
	) ([]*AgentSafetySubject, error)
	Assess(ctx context.Context, req *AssessAgentSafetyRequest) []AgentToolSafety
	ReachWarnings(req *AgentReachRequest) []AgentReachWarning
}
