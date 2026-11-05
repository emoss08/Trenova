package scimports

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Directory struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	TokenID        pulid.ID
	TenantSlug     string
}

func (d *Directory) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: d.OrganizationID, BuID: d.BusinessUnitID}
}

func (d *Directory) DBTenant() dbscope.Tenant {
	return dbscope.Tenant{OrganizationID: d.OrganizationID, BusinessUnitID: d.BusinessUnitID}
}

type Credential struct {
	TokenID          pulid.ID
	DirectoryID      pulid.ID
	OrganizationID   pulid.ID
	BusinessUnitID   pulid.ID
	TokenHash        string
	Status           iam.SCIMTokenStatus
	ExpiresAt        int64
	LastUsedAt       int64
	DirectoryEnabled bool
	TenantSlug       string
}

type AuthenticateRequest struct {
	Token      string
	TenantSlug string
}
