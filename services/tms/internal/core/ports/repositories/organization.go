package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetOrganizationByIDRequest struct {
	TenantInfo   pagination.TenantInfo
	IncludeState bool `json:"includeState"`
	IncludeBU    bool `json:"includeBu"`
}

type GetOrganizationsByIDsRequest struct {
	TenantInfo      pagination.TenantInfo
	OrganizationIDs []pulid.ID
	IncludeState    bool `json:"includeState"`
	IncludeBU       bool `json:"includeBu"`
}

type SelectOrganizationOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
	Scope              string `json:"scope"`
	ExcludeCurrent     bool   `json:"excludeCurrent"`
}

// BrokerageDependencyCounts reports outstanding brokerage work for an
// organization. It backs the guard that refuses to turn brokerage off while
// any of these remain open.
type BrokerageDependencyCounts struct {
	ActiveTenders             int `json:"activeTenders"`
	ActiveRateConfirmations   int `json:"activeRateConfirmations"`
	UnpaidCarrierSettlements  int `json:"unpaidCarrierSettlements"`
	OpenCarrierInvoiceMatches int `json:"openCarrierInvoiceMatches"`
	ActiveCarrierAssignments  int `json:"activeCarrierAssignments"`
}

func (c BrokerageDependencyCounts) HasOutstandingWork() bool {
	return c.ActiveTenders > 0 ||
		c.ActiveRateConfirmations > 0 ||
		c.UnpaidCarrierSettlements > 0 ||
		c.OpenCarrierInvoiceMatches > 0 ||
		c.ActiveCarrierAssignments > 0
}

type AssetDependencyCounts struct {
	ActiveDriverAssignments int `json:"activeDriverAssignments"`
	UnpaidDriverSettlements int `json:"unpaidDriverSettlements"`
	LinkedPortalUsers       int `json:"linkedPortalUsers"`
}

func (c AssetDependencyCounts) HasOutstandingWork() bool {
	return c.ActiveDriverAssignments > 0 ||
		c.UnpaidDriverSettlements > 0 ||
		c.LinkedPortalUsers > 0
}

type CapabilityLock string

const (
	CapabilityLockNone   = CapabilityLock("")
	CapabilityLockShare  = CapabilityLock("SHARE")
	CapabilityLockUpdate = CapabilityLock("UPDATE")
)

type GetOrganizationCapabilitiesRequest struct {
	TenantInfo pagination.TenantInfo
	Lock       CapabilityLock
}

type OrganizationCapabilities struct {
	BrokerageEnabled       bool `json:"brokerageEnabled"`
	AssetOperationsEnabled bool `json:"assetOperationsEnabled"`
}

type OrganizationRepository interface {
	GetByID(ctx context.Context, req GetOrganizationByIDRequest) (*tenant.Organization, error)
	GetCapabilities(
		ctx context.Context,
		req GetOrganizationCapabilitiesRequest,
	) (*OrganizationCapabilities, error)
	GetByIDs(ctx context.Context, req GetOrganizationsByIDsRequest) ([]*tenant.Organization, error)
	SelectOptions(
		ctx context.Context,
		req *SelectOrganizationOptionsRequest,
	) (*pagination.ListResult[*tenant.Organization], error)
	GetByLoginSlug(ctx context.Context, loginSlug string) (*tenant.Organization, error)
	ListLoginSlugsByPrefix(ctx context.Context, prefix string) ([]string, error)
	Update(ctx context.Context, entity *tenant.Organization) (*tenant.Organization, error)
	ClearLogoURL(ctx context.Context, orgID pulid.ID, version int64) (*tenant.Organization, error)
	CountBrokerageDependencies(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*BrokerageDependencyCounts, error)
	CountAssetDependencies(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*AssetDependencyCounts, error)
}

type OrganizationCacheRepository interface {
	GetByID(ctx context.Context, orgID pulid.ID) (*tenant.Organization, error)
}
