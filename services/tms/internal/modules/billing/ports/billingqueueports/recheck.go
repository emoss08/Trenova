package billingqueueports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RecheckRequester interface {
	RequestRecheck(ctx context.Context, tenantInfo pagination.TenantInfo, itemIDs ...pulid.ID)
}

type RecheckCandidate struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (c *RecheckCandidate) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: c.OrganizationID, BuID: c.BusinessUnitID}
}

type ListRecheckCandidatesRequest struct {
	StaleBefore int64
	Limit       int
}

type RecheckResult struct {
	ItemID pulid.ID            `json:"itemId"`
	Status billingqueue.Status `json:"status"`
	Ready  bool                `json:"ready"`
}
