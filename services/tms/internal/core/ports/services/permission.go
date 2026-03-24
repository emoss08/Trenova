package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

type PermissionCheckRequest struct {
	PrincipalType  PrincipalType
	PrincipalID    pulid.ID
	UserID         pulid.ID
	APIKeyID       pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	Resource       string
	Operation      permission.Operation
	ResourceID     *pulid.ID
}

type PermissionCheckResult struct {
	Allowed       bool
	Reason        string
	DataScope     permission.DataScope
	CacheHit      bool
	CheckDuration int64
}

type BatchPermissionCheckRequest struct {
	PrincipalType  PrincipalType
	PrincipalID    pulid.ID
	UserID         pulid.ID
	APIKeyID       pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	Checks         []ResourceOperationCheck
}

type ResourceOperationCheck struct {
	Resource   string
	Operation  permission.Operation
	ResourceID *pulid.ID
}

type BatchPermissionCheckResult struct {
	Results       []PermissionCheckResult
	CacheHit      bool
	CheckDuration int64
}

type LightPermissionManifest struct {
	Version             string                      `json:"version"`
	UserID              pulid.ID                    `json:"userId"`
	OrganizationID      pulid.ID                    `json:"organizationId"`
	IsPlatformAdmin     bool                        `json:"isPlatformAdmin"`
	IsOrgAdmin          bool                        `json:"isOrgAdmin"`
	IsBusinessUnitAdmin bool                        `json:"isBusinessUnitAdmin"`
	MaxSensitivity      permission.FieldSensitivity `json:"maxSensitivity"`
	Permissions         map[string]uint32           `json:"permissions"`
	RouteAccess         map[string]bool             `json:"routeAccess"`
	AvailableOrgs       []OrgSummary                `json:"availableOrgs"`
	Checksum            string                      `json:"checksum"`
	ExpiresAt           int64                       `json:"expiresAt"`
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
	ID                  pulid.ID `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	IsSystem            bool     `json:"isSystem"`
	IsOrgAdmin          bool     `json:"isOrgAdmin"`
	IsBusinessUnitAdmin bool     `json:"isBusinessUnitAdmin"`
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
}
