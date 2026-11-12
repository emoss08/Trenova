package billingqueueports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RecheckRequester asks for an item's billing checks to run again. It marks
// the items due in the caller's transaction, so a request is never lost
// between the commit and the signal, and wakes the tenant's drain after
// commit.
type RecheckRequester interface {
	RequestRecheck(ctx context.Context, tenantInfo pagination.TenantInfo, itemIDs ...pulid.ID) error
	// WakeTenant wakes the tenant's drain without marking anything: the
	// sweep's backstop for a signal that was lost.
	WakeTenant(ctx context.Context, tenantInfo pagination.TenantInfo)
}

type MarkRecheckDueRequest struct {
	TenantInfo pagination.TenantInfo
	ItemIDs    []pulid.ID
	At         int64
}

type ListRecheckDueRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
	Limit      int
}

type RecheckDueItem struct {
	ID    pulid.ID
	DueAt int64
}

// SettleRecheckRequest clears the item's due marker, or moves it to RetryAt,
// only while it still holds ClaimedDueAt. A request moves the marker
// forward, so one that landed while the checks ran keeps the item due.
type SettleRecheckRequest struct {
	TenantInfo   pagination.TenantInfo
	ItemID       pulid.ID
	ClaimedDueAt int64
	RetryAt      *int64
}

type MarkStaleRecheckDueRequest struct {
	StaleBefore int64
	Now         int64
	Limit       int
}

type ListRecheckDueTenantsRequest struct {
	Now   int64
	Limit int
}

type RecheckResult struct {
	ItemID pulid.ID            `json:"itemId"`
	Status billingqueue.Status `json:"status"`
	Ready  bool                `json:"ready"`
}
