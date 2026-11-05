package scimports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
)

const (
	FilterID          = "id"
	FilterUserName    = "username"
	FilterExternalID  = "externalid"
	FilterEmail       = "emails.value"
	FilterActive      = "active"
	FilterDisplayName = "displayname"
	FilterMember      = "members.value"
)

func UserFilters() []string {
	return []string{FilterID, FilterUserName, FilterExternalID, FilterEmail, FilterActive}
}

func GroupFilters() []string {
	return []string{FilterID, FilterDisplayName, FilterExternalID, FilterMember}
}

type ListQuery struct {
	DirectoryID pulid.ID
	Filter      *scimproto.Filter
	Offset      int
	Limit       int
}

type UserRecord struct {
	Link *scim.User
	User *tenant.User
}

type MemberRecord struct {
	UserID pulid.ID
	Name   string
}

type MemberChange struct {
	DirectoryID pulid.ID
	GroupID     pulid.ID
	UserIDs     []pulid.ID
}

// MappingFilter picks the group role mappings whose grants need a fresh
// look: those one person saved, those in one organization, or both.
type MappingFilter struct {
	OrganizationID pulid.ID
	SavedByID      pulid.ID
}

type PendingDirectory struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	DirectoryID    pulid.ID
}

type CredentialRepository interface {
	FindCredential(ctx context.Context, prefix string) (*Credential, error)
	TouchCredential(ctx context.Context, tokenID pulid.ID, now int64) error
}

type UserRepository interface {
	ListUsers(ctx context.Context, q *ListQuery) ([]*UserRecord, int, error)
	GetUser(ctx context.Context, directoryID, userID pulid.ID) (*UserRecord, error)
	FindMember(ctx context.Context, emailAddress string) (*tenant.User, error)
	CreateUser(ctx context.Context, link *scim.User) error
	SaveUser(ctx context.Context, link *scim.User) error
	DeleteUser(ctx context.Context, directoryID, userID pulid.ID) error
	CountUsers(ctx context.Context, directoryID pulid.ID, userIDs []pulid.ID) (int, error)
}

type GroupRepository interface {
	ListGroups(ctx context.Context, q *ListQuery) ([]*scim.Group, int, error)
	GetGroup(ctx context.Context, directoryID, groupID pulid.ID) (*scim.Group, error)
	CreateGroup(ctx context.Context, group *scim.Group) error
	SaveGroup(ctx context.Context, group *scim.Group) error
	DeleteGroup(ctx context.Context, directoryID, groupID pulid.ID) error
	ListMembers(
		ctx context.Context,
		directoryID, groupID pulid.ID,
		limit int,
	) ([]*MemberRecord, error)
	AddMembers(ctx context.Context, change *MemberChange) error
	RemoveMembers(ctx context.Context, change *MemberChange) error
	ClearMembers(ctx context.Context, directoryID, groupID pulid.ID) error
}

type GrantRepository interface {
	MarkPending(ctx context.Context, directoryID pulid.ID, userIDs []pulid.ID) error
	MarkGroupPending(ctx context.Context, directoryID, groupID pulid.ID) error
	MarkOrganizationPending(ctx context.Context) ([]pulid.ID, error)
	MarkMappingsPending(ctx context.Context, filter MappingFilter) ([]PendingDirectory, error)
	MarkExpiredSaversPending(ctx context.Context, since, until int64) error
	ListPending(ctx context.Context, directoryID pulid.ID, limit int) ([]*scim.User, error)
	ListPendingDirectories(ctx context.Context, limit int) ([]PendingDirectory, error)
	CompleteReconcile(ctx context.Context, link *scim.User) error
	RoleSources(ctx context.Context, directoryID, userID pulid.ID) ([]*scim.RoleSource, error)
	ListGrants(ctx context.Context, directoryID, userID pulid.ID) ([]*scim.RoleGrant, error)
	CreateGrant(ctx context.Context, grant *scim.RoleGrant) error
	DeleteDirectoryAssignments(
		ctx context.Context,
		directoryID pulid.ID,
	) ([]*rbac.UserRoleAssignment, error)
}

type ProvisioningRecorder interface {
	RecordProvisioning(ctx context.Context, record *iam.ProvisioningAuditRecord) error
}

type Repository interface {
	CredentialRepository
	UserRepository
	GroupRepository
	GrantRepository
	ProvisioningRecorder
}
