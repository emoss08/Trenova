package temporaljobs

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultTenantScanLimit        = 100
	DefaultTenantRecordLimit      = 100
	DefaultTenantDispatchLimit    = 4
	DefaultAuditTenantBatchSize   = 500
	DefaultAuditFlushEntriesLimit = 5000
)

type TenantWorkItem struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Cursor         string   `json:"cursor,omitempty"`
	Count          int      `json:"count,omitempty"`
	Limit          int      `json:"limit,omitempty"`
}

func NewTenantWorkItem(tenantInfo pagination.TenantInfo, limit int) TenantWorkItem {
	return TenantWorkItem{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Limit:          NormalizeLimit(limit, DefaultTenantRecordLimit),
	}
}

func (i TenantWorkItem) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: i.OrganizationID,
		BuID:  i.BusinessUnitID,
	}
}

type TenantPartialFailure struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Error          string   `json:"error"`
}

type TenantRunResult struct {
	TenantsScanned   int                    `json:"tenantsScanned"`
	TenantsProcessed int                    `json:"tenantsProcessed"`
	RecordsProcessed int                    `json:"recordsProcessed"`
	SkippedCount     int                    `json:"skippedCount"`
	FailureCount     int                    `json:"failureCount"`
	PartialFailures  []TenantPartialFailure `json:"partialFailures,omitempty"`
}

func (r *TenantRunResult) AddTenantResult(processed, skipped int) {
	r.TenantsProcessed++
	r.RecordsProcessed += processed
	r.SkippedCount += skipped
}

func (r *TenantRunResult) AddFailure(item TenantWorkItem, err error) {
	r.FailureCount++
	failure := TenantPartialFailure{
		OrganizationID: item.OrganizationID,
		BusinessUnitID: item.BusinessUnitID,
	}
	if err != nil {
		failure.Error = err.Error()
	}
	r.PartialFailures = append(r.PartialFailures, failure)
}

func NormalizeLimit(limit, fallback int) int {
	if limit > 0 {
		return limit
	}
	return fallback
}

func BuildTenantWorkItems(tenants []pagination.TenantInfo, limit int) []TenantWorkItem {
	items := make([]TenantWorkItem, 0, len(tenants))
	for _, tenantInfo := range tenants {
		items = append(items, NewTenantWorkItem(tenantInfo, limit))
	}
	return items
}
