package scimports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
)

const (
	MaxPageSize          = 100
	MaxPatchOperations   = 100
	MaxMembersPerRequest = 1000
	MaxMembersReturned   = 1000
)

type ListRequest struct {
	Directory  *Directory
	Filter     string
	StartIndex int
	Count      int
}

type UserPage struct {
	Users      []*scimproto.User
	Total      int
	StartIndex int
}

type GroupPage struct {
	Groups     []*scimproto.Group
	Total      int
	StartIndex int
}

type GetGroupRequest struct {
	Directory      *Directory
	GroupID        pulid.ID
	IncludeMembers bool
}

type Provisioner interface {
	Authenticate(ctx context.Context, req AuthenticateRequest) (*Directory, error)

	ListUsers(ctx context.Context, req *ListRequest) (*UserPage, error)
	GetUser(ctx context.Context, dir *Directory, userID pulid.ID) (*scimproto.User, error)
	CreateUser(
		ctx context.Context,
		dir *Directory,
		resource *scimproto.User,
	) (*scimproto.User, error)
	ReplaceUser(
		ctx context.Context,
		dir *Directory,
		userID pulid.ID,
		resource *scimproto.User,
	) (*scimproto.User, error)
	PatchUser(
		ctx context.Context,
		dir *Directory,
		userID pulid.ID,
		operations []scimproto.PatchOperation,
	) (*scimproto.User, error)
	DeleteUser(ctx context.Context, dir *Directory, userID pulid.ID) error

	ListGroups(ctx context.Context, req *ListRequest) (*GroupPage, error)
	GetGroup(ctx context.Context, req *GetGroupRequest) (*scimproto.Group, error)
	CreateGroup(
		ctx context.Context,
		dir *Directory,
		resource *scimproto.Group,
	) (*scimproto.Group, error)
	ReplaceGroup(
		ctx context.Context,
		dir *Directory,
		groupID pulid.ID,
		resource *scimproto.Group,
	) (*scimproto.Group, error)
	PatchGroup(
		ctx context.Context,
		dir *Directory,
		groupID pulid.ID,
		operations []scimproto.PatchOperation,
	) error
	DeleteGroup(ctx context.Context, dir *Directory, groupID pulid.ID) error
}

type ReconcileResult struct {
	Reconciled int  `json:"reconciled"`
	Failed     int  `json:"failed"`
	More       bool `json:"more"`
}

type GrantReconciler interface {
	ReconcilePending(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		directoryID pulid.ID,
	) (*ReconcileResult, error)
	RequestPendingReconciles(ctx context.Context) (int, error)
}

type ReconcileRequester interface {
	RequestReconcile(ctx context.Context, tenantInfo pagination.TenantInfo, directoryID pulid.ID)
}

type DirectoryGrants interface {
	MappingsChanged(ctx context.Context, tenantInfo pagination.TenantInfo) error
	RevokeDirectoryGrants(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		directoryID pulid.ID,
	) error
}
