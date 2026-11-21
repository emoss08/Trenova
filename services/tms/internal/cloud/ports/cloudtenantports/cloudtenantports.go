package cloudtenantports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrNotFound = errors.New("no tenant by that instance is on this server")
	ErrNotHeld  = errors.New("the organization is not a tenant of this shared server")
)

type ListRequest struct {
	After string
	Limit int
}

type Page struct {
	Tenants []*cloudtenant.Tenant
	Next    string
}

type Repository interface {
	Lock(ctx context.Context, instanceID string) error
	Get(ctx context.Context, instanceID string) (*cloudtenant.Tenant, error)
	Save(ctx context.Context, tenant *cloudtenant.Tenant) error
	List(ctx context.Context, req *ListRequest) (*Page, error)
	ByInstances(ctx context.Context, instanceIDs []string) ([]*cloudtenant.Tenant, error)
	InstanceOf(ctx context.Context, organizationID pulid.ID) (string, error)
	Record(ctx context.Context, event *cloudtenant.Event) error
}

type AdministratorFacts struct {
	UserID      pulid.ID
	SetUp       bool
	InviteLive  bool
	InviteToken string
}

type AdministratorRequest struct {
	UserID      pulid.ID
	InviteToken string
}

type Administrators interface {
	Facts(
		ctx context.Context,
		requests []AdministratorRequest,
	) (map[pulid.ID]AdministratorFacts, error)
	EmailTaken(ctx context.Context, email string) (bool, error)
}

type Member struct {
	UserID           pulid.ID
	OtherMemberships int
}

type MembersPage struct {
	Members []Member
	Next    pulid.ID
}

type Members interface {
	Page(
		ctx context.Context,
		tenant dbscope.Tenant,
		after pulid.ID,
		limit int,
	) (*MembersPage, error)
	Reassign(ctx context.Context, tenant dbscope.Tenant, userID pulid.ID) (bool, error)
}

type PurgeTableOutcome struct {
	Table   string `json:"table"`
	Deleted int64  `json:"deleted"`
}

type PurgeRowsRequest struct {
	Tenant     dbscope.Tenant
	BatchSize  int
	MaxBatches int
	Beat       func()
}

type PurgeRowsResult struct {
	Deleted  int64               `json:"deleted"`
	Tables   []PurgeTableOutcome `json:"tables"`
	Left     []string            `json:"left"`
	Complete bool                `json:"complete"`
}

type DeleteResult struct {
	Deleted   bool   `json:"deleted"`
	BlockedBy string `json:"blockedBy,omitempty"`
}

type TableCount struct {
	Table string `json:"table" db:"table_name"`
	Rows  int64  `json:"rows"  db:"row_count"`
}

type Purge interface {
	PurgeRows(ctx context.Context, req *PurgeRowsRequest) (*PurgeRowsResult, error)
	DeleteOrganization(ctx context.Context, tenant dbscope.Tenant) (*DeleteResult, error)
	DeleteBusinessUnit(ctx context.Context, tenant dbscope.Tenant) (*DeleteResult, error)
	Remaining(ctx context.Context, tenant dbscope.Tenant) ([]TableCount, error)
}

type KeyPurge interface {
	DeleteMatching(ctx context.Context, fragment string) (int64, error)
}

type SearchPurge interface {
	DeleteOrganizationDocuments(ctx context.Context, organizationID pulid.ID) (int, error)
}

type LiveSupportGrants interface {
	ListLive(
		ctx context.Context,
		req *supportaccessports.ListLiveRequest,
	) ([]*supportaccessports.LiveGrant, error)
}

var (
	ErrCommandNotFound = errors.New("no command by that id is on this server")
	ErrGrantMissing    = errors.New(
		"the server holds no unexpired nightly archive location; the tenant agent delivers it",
	)
)

type Commands interface {
	Get(ctx context.Context, commandID string) (*tenantcommand.Command, error)
	Create(ctx context.Context, cmd *tenantcommand.Command) (*tenantcommand.Command, bool, error)
	Save(ctx context.Context, cmd *tenantcommand.Command) error
	Open(ctx context.Context, instanceID string) ([]*tenantcommand.Command, error)
	Failures(ctx context.Context, instanceIDs []string) ([]*tenantcommand.Command, error)
	Progress(ctx context.Context, commandID string, progress *tenantcommand.Progress) error
	SetGrant(ctx context.Context, grant *tenantcommand.Grant) error
	Grant(ctx context.Context, name string) (*tenantcommand.Grant, error)
}
