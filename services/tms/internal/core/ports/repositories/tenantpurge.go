package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TenantMember struct {
	UserID           pulid.ID `json:"userId"`
	Name             string   `json:"name"`
	EmailAddress     string   `json:"emailAddress"`
	Username         string   `json:"username"`
	OtherMemberships int      `json:"otherMemberships"`
}

type TenantProfile struct {
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}

type PurgeTenantRowsRequest struct {
	TenantInfo pagination.TenantInfo
	BatchSize  int
	MaxBatches int
}

type PurgeTableOutcome struct {
	Table   string `json:"table"`
	Deleted int64  `json:"deleted"`
}

type PurgeTenantRowsResult struct {
	Deleted  int64               `json:"deleted"`
	Tables   []PurgeTableOutcome `json:"tables"`
	Retained []string            `json:"retained"`
	Blocked  []string            `json:"blocked"`
	Complete bool                `json:"complete"`
}

type PurgeTenantUserRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

type PurgeTenantUserResult struct {
	Deleted     bool `json:"deleted"`
	Deactivated bool `json:"deactivated"`
	Reassigned  bool `json:"reassigned"`
}

type DeleteTenantResult struct {
	SignupsDeleted      int64  `json:"signupsDeleted"`
	OrganizationDeleted bool   `json:"organizationDeleted"`
	BusinessUnitDeleted bool   `json:"businessUnitDeleted"`
	RetainedReason      string `json:"retainedReason,omitempty"`
}

type TenantPurgeRepository interface {
	ListMembers(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*TenantMember, error)
	OrganizationProfile(ctx context.Context, tenantInfo pagination.TenantInfo) (*TenantProfile, error)
	PurgeRows(ctx context.Context, req *PurgeTenantRowsRequest) (*PurgeTenantRowsResult, error)
	PurgeUser(ctx context.Context, req *PurgeTenantUserRequest) (*PurgeTenantUserResult, error)
	DeleteTenant(ctx context.Context, tenantInfo pagination.TenantInfo) (*DeleteTenantResult, error)
}
