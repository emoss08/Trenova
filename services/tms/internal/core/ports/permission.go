package ports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pulid"
)

type PermissionEngine interface {
	Check(ctx context.Context, req *PermissionCheckRequest) (*PermissionCheckResult, error)
	CheckBatch(
		ctx context.Context,
		req *BatchPermissionCheckRequest,
	) (*BatchPermissionCheckResult, error)
	GetUserPermissions(
		ctx context.Context,
		userID, organizationID pulid.ID,
	) (*PermissionManifest, error)
	RefreshUserPermissions(ctx context.Context, userID, organizationID pulid.ID) error
	InvalidateCache(ctx context.Context, userID, organizationID pulid.ID) error
	HasAdminRole(ctx context.Context, userID, organizationID pulid.ID) (bool, error)
}

type PolicyRepository interface {
	GetByID(ctx context.Context, id pulid.ID) (*permission.Policy, error)
	GetByBusinessUnit(ctx context.Context, businessUnitID pulid.ID) ([]*permission.Policy, error)
	GetByOrganization(
		ctx context.Context,
		businessUnitID, organizationID pulid.ID,
	) ([]*permission.Policy, error)
	GetUserPolicies(
		ctx context.Context,
		userID, organizationID pulid.ID,
	) ([]*permission.Policy, error)
	Create(ctx context.Context, policy *permission.Policy) error
	Update(ctx context.Context, policy *permission.Policy) error
	Delete(ctx context.Context, id pulid.ID) error
	GetResourcePolicies(
		ctx context.Context,
		businessUnitID pulid.ID,
		resourceType string,
	) ([]*permission.Policy, error)
}

type RoleRepository interface {
	GetByID(ctx context.Context, id pulid.ID) (*permission.Role, error)
	GetByBusinessUnit(ctx context.Context, businessUnitID pulid.ID) ([]*permission.Role, error)
	GetUserRoles(ctx context.Context, userID, organizationID pulid.ID) ([]*permission.Role, error)
	HasAdminRole(ctx context.Context, userID, organizationID pulid.ID) (bool, error)
	Create(ctx context.Context, role *permission.Role) error
	Update(ctx context.Context, role *permission.Role) error
	Delete(ctx context.Context, id pulid.ID) error
	AssignToUser(
		ctx context.Context,
		userID, organizationID, roleID pulid.ID,
		assignedBy pulid.ID,
		expiresAt *time.Time,
	) error
	RemoveFromUser(ctx context.Context, userID, organizationID, roleID pulid.ID) error
	GetRoleHierarchy(ctx context.Context, roleID pulid.ID) ([]*permission.Role, error)
}

type PermissionCacheRepository interface {
	Get(ctx context.Context, userID, organizationID pulid.ID) (*CachedPermissions, error)
	Set(
		ctx context.Context,
		userID, organizationID pulid.ID,
		permissions *CachedPermissions,
		ttl time.Duration,
	) error
	Delete(ctx context.Context, userID, organizationID pulid.ID) error
	DeletePattern(pattern string) error
	Exists(ctx context.Context, userID, organizationID pulid.ID) (bool, error)
	GetVersion(ctx context.Context, userID, organizationID pulid.ID) (string, error)
}

type PolicyCompiler interface {
	Compile(policies []*permission.Policy) (*CompiledPermissions, error)
	CompileForUser(
		userID, organizationID pulid.ID,
		policies []*permission.Policy,
	) (*CompiledPermissions, error)
	OptimizeBitfields(actions []string) uint32
	BuildBloomFilter(permissions *CompiledPermissions) ([]byte, error)
}

type PermissionCheckRequest struct {
	UserID         pulid.ID              `json:"userId"`
	OrganizationID pulid.ID              `json:"organizationId"`
	ResourceType   string                `json:"resourceType"`
	Action         string                `json:"action"`
	ResourceID     *pulid.ID             `json:"resourceId,omitempty"`
	Context        map[string]any        `json:"context,omitempty"`
	DataScope      *permission.DataScope `json:"dataScope,omitempty"`
}

type PermissionCheckResult struct {
	Allowed       bool                 `json:"allowed"`
	Reason        string               `json:"reason,omitempty"`
	DataScope     permission.DataScope `json:"dataScope"`
	AppliedPolicy *permission.Policy   `json:"appliedPolicy,omitempty"`
	CacheHit      bool                 `json:"cacheHit"`
	ComputeTimeMs float64              `json:"computeTimeMs"`
}

type BatchPermissionCheckRequest struct {
	UserID         pulid.ID                  `json:"userId"`
	OrganizationID pulid.ID                  `json:"organizationId"`
	Checks         []*PermissionCheckRequest `json:"checks"`
}

type BatchPermissionCheckResult struct {
	Results      []*PermissionCheckResult `json:"results"`
	CacheHitRate float64                  `json:"cacheHitRate"`
	TotalTimeMs  float64                  `json:"totalTimeMs"`
}

type PermissionManifest struct {
	Version       string                `json:"version"`
	UserID        pulid.ID              `json:"userId"`
	CurrentOrg    pulid.ID              `json:"currentOrg"`
	AvailableOrgs []pulid.ID            `json:"availableOrgs"`
	ComputedAt    time.Time             `json:"computedAt"`
	ExpiresAt     time.Time             `json:"expiresAt"`
	Resources     ResourcePermissionMap `json:"resources"`
	BloomFilter   []byte                `json:"bloomFilter"`
	Checksum      string                `json:"checksum"`
}

type ResourcePermissionMap map[string]*ResourcePermission

type ResourcePermission struct {
	StandardOps uint32               `json:"standardOps"`
	ExtendedOps []string             `json:"extendedOps,omitempty"`
	DataScope   permission.DataScope `json:"dataScope"`
	QuickCheck  uint64               `json:"quickCheck"`
}

type CachedPermissions struct {
	Version     string               `json:"version"`
	ComputedAt  time.Time            `json:"computedAt"`
	ExpiresAt   time.Time            `json:"expiresAt"`
	Permissions *CompiledPermissions `json:"permissions"`
	BloomFilter []byte               `json:"bloomFilter"`
	Checksum    string               `json:"checksum"`
}

type CompiledPermissions struct {
	Resources   ResourcePermissionMap           `json:"resources"`
	GlobalFlags uint64                          `json:"globalFlags"`
	DataScopes  map[string]permission.DataScope `json:"dataScopes"`
}

const (
	ActionCreate uint32 = 1 << iota
	ActionRead
	ActionUpdate
	ActionDelete
	ActionList
	ActionExport
	ActionImport
	ActionApprove
	ActionReject
	ActionArchive
)

var ActionBits = map[string]uint32{
	"create":  ActionCreate,
	"read":    ActionRead,
	"update":  ActionUpdate,
	"delete":  ActionDelete,
	"list":    ActionList,
	"export":  ActionExport,
	"import":  ActionImport,
	"approve": ActionApprove,
	"reject":  ActionReject,
	"archive": ActionArchive,
}

func HasAction(bitfield uint32, action string) bool {
	if bit, ok := ActionBits[action]; ok {
		return (bitfield & bit) != 0
	}
	return false
}

func AddAction(bitfield uint32, action string) uint32 {
	if bit, ok := ActionBits[action]; ok {
		return bitfield | bit
	}
	return bitfield
}

func RemoveAction(bitfield uint32, action string) uint32 {
	if bit, ok := ActionBits[action]; ok {
		return bitfield &^ bit
	}
	return bitfield
}
