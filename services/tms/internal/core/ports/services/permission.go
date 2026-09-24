package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

type PermissionCheckRequest struct {
	PrincipalType      PrincipalType
	PrincipalID        pulid.ID
	UserID             pulid.ID
	APIKeyID           pulid.ID
	BusinessUnitID     pulid.ID
	OrganizationID     pulid.ID
	Resource           string
	Operation          permission.Operation
	ResourceID         *pulid.ID
	ResourceAttributes ResourceAttributes
	ContextAttributes  RequestContextAttributes
}

type PermissionCheckResult struct {
	Allowed       bool
	Reason        string
	DataScope     permission.DataScope
	CacheHit      bool
	CheckDuration int64
}

type BatchPermissionCheckRequest struct {
	PrincipalType     PrincipalType
	PrincipalID       pulid.ID
	UserID            pulid.ID
	APIKeyID          pulid.ID
	BusinessUnitID    pulid.ID
	OrganizationID    pulid.ID
	Checks            []ResourceOperationCheck
	ContextAttributes RequestContextAttributes
}

type ResourceOperationCheck struct {
	Resource           string
	Operation          permission.Operation
	ResourceID         *pulid.ID
	ResourceAttributes ResourceAttributes
}

type ResourceAttributes struct {
	OrganizationID pulid.ID `json:"organizationId,omitempty"`
	BusinessUnitID pulid.ID `json:"businessUnitId,omitempty"`
	OwnerID        pulid.ID `json:"ownerId,omitempty"`
	TerminalID     pulid.ID `json:"terminalId,omitempty"`
	ActiveRoleID   pulid.ID `json:"activeRoleId,omitempty"`
}

type RequestContextAttributes struct {
	ActiveRoleIDs         []pulid.ID `json:"activeRoleIds,omitempty"`
	AuthenticatorAAL      int        `json:"authenticatorAal,omitempty"`
	FederationFAL         int        `json:"federationFal,omitempty"`
	MFAAuthenticatedAt    int64      `json:"mfaAuthenticatedAt,omitempty"`
	LastReauthenticatedAt int64      `json:"lastReauthenticatedAt,omitempty"`
	RiskDecision          string     `json:"riskDecision,omitempty"`
}

type BatchPermissionCheckResult struct {
	Results       []PermissionCheckResult
	CacheHit      bool
	CheckDuration int64
}

type LightPermissionManifest struct {
	Version                string                      `json:"version"`
	UserID                 pulid.ID                    `json:"userId"`
	OrganizationID         pulid.ID                    `json:"organizationId"`
	ActiveRoleIDs          []pulid.ID                  `json:"activeRoleIds"`
	AuthorizedRoleIDs      []pulid.ID                  `json:"authorizedRoleIds"`
	ActiveRoles            []RoleSummary               `json:"activeRoles"`
	AuthorizedRoles        []RoleSummary               `json:"authorizedRoles"`
	RequiresRoleActivation bool                        `json:"requiresRoleActivation"`
	MaxSensitivity         permission.FieldSensitivity `json:"maxSensitivity"`
	Permissions            map[string]uint32           `json:"permissions"`
	RouteAccess            map[string]bool             `json:"routeAccess"`
	AvailableOrgs          []OrgSummary                `json:"availableOrgs"`
	Checksum               string                      `json:"checksum"`
	ExpiresAt              int64                       `json:"expiresAt"`
}

type OrgSummary struct {
	ID   pulid.ID `json:"id"`
	Name string   `json:"name"`
}

type ResourcePermissionDetail struct {
	Resource         string                      `json:"resource"`
	Operations       []permission.Operation      `json:"operations"`
	DataScope        permission.DataScope        `json:"dataScope"`
	MaxSensitivity   permission.FieldSensitivity `json:"maxSensitivity"`
	AccessibleFields []string                    `json:"accessibleFields"`
}

type EffectivePermissions struct {
	UserID         pulid.ID                               `json:"userId"`
	OrganizationID pulid.ID                               `json:"organizationId"`
	Roles          []RoleSummary                          `json:"roles"`
	MaxSensitivity permission.FieldSensitivity            `json:"maxSensitivity"`
	Resources      map[string]EffectiveResourcePermission `json:"resources"`
}

type RoleSummary struct {
	ID              pulid.ID `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	IsSystem        bool     `json:"isSystem"`
	PermissionCount int      `json:"permissionCount"`
}

func NewRoleSummary(role *permission.Role) RoleSummary {
	return RoleSummary{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
	}
}

func ApplyRolePermissionCounts(summaries []RoleSummary, counts map[pulid.ID]int) {
	for i := range summaries {
		summaries[i].PermissionCount = counts[summaries[i].ID]
	}
}

type EffectiveResourcePermission struct {
	Operations []permission.Operation `json:"operations"`
	DataScope  permission.DataScope   `json:"dataScope"`
	GrantedBy  []string               `json:"grantedBy"`
}

type SimulatePermissionsRequest struct {
	UserID         pulid.ID
	OrganizationID pulid.ID
	AddRoleIDs     []pulid.ID
	RemoveRoleIDs  []pulid.ID
}

type PermissionEngine interface {
	Check(ctx context.Context, req *PermissionCheckRequest) (*PermissionCheckResult, error)
	CheckBatch(
		ctx context.Context,
		req *BatchPermissionCheckRequest,
	) (*BatchPermissionCheckResult, error)
	GetLightManifest(ctx context.Context, userID, orgID pulid.ID) (*LightPermissionManifest, error)
	GetResourcePermissions(
		ctx context.Context,
		userID, orgID pulid.ID,
		resource string,
	) (*ResourcePermissionDetail, error)
	InvalidateUser(ctx context.Context, userID, orgID pulid.ID) error
	GetEffectivePermissions(
		ctx context.Context,
		userID, orgID pulid.ID,
	) (*EffectivePermissions, error)
	SimulatePermissions(
		ctx context.Context,
		req *SimulatePermissionsRequest,
	) (*EffectivePermissions, error)
	// AgentsUsable is what the actor may use of the organization's agents:
	// whether they hold the assistant at the operation, and the agents their
	// roles, inherited ones included, grant.
	AgentsUsable(
		ctx context.Context,
		actor *RequestActor,
		operation permission.Operation,
	) (*UsableAgents, error)
	// MayUseAgent reports whether the actor may start or continue a
	// conversation with the agent.
	MayUseAgent(
		ctx context.Context,
		actor *RequestActor,
		definition *agentdefinition.Definition,
	) (bool, error)
	// RoleCoverage says, for each role, how much of what an agent's tools
	// need the role grants through itself and the roles it inherits.
	RoleCoverage(ctx context.Context, req *RoleCoverageRequest) ([]RoleCoverage, error)
}

// UsableAgents is what a person may use of the organization's agents.
type UsableAgents struct {
	Assistant  bool
	GrantedIDs []pulid.ID
}

// Allows reports whether the person may use the agent.
func (u *UsableAgents) Allows(definition *agentdefinition.Definition) bool {
	return u != nil && u.Assistant && definition != nil &&
		definition.UsableWith(u.GrantedIDs)
}

// Audience narrows a read to the agents the person may use.
func (u *UsableAgents) Audience() *repositories.AgentAudience {
	if u == nil {
		return &repositories.AgentAudience{}
	}

	return &repositories.AgentAudience{GrantedAgentIDs: u.GrantedIDs}
}

// RequiredGrant is a resource and operation a tool needs of the person
// using it.
type RequiredGrant struct {
	Tool      string
	Resource  permission.Resource
	Operation permission.Operation
}

type RoleCoverageRequest struct {
	OrganizationID pulid.ID
	RoleIDs        []pulid.ID
	Required       []RequiredGrant
}

type RoleCoverage struct {
	RoleID           pulid.ID
	Coverage         agentdefinition.AudienceCoverage
	MissingResources []permission.Resource
}
