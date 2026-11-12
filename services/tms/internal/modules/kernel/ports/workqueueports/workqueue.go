package workqueueports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// Queue holds the subjects waiting for background work, per tenant and kind.
// A request is a row written in the requester's transaction; a tenant's drain
// reads its due rows a page at a time and settles each.
type Queue interface {
	// Enqueue makes the subjects due. Every request moves a subject's due
	// time strictly forward, so a request that lands while the subject's
	// work runs survives the settle.
	Enqueue(ctx context.Context, req *EnqueueRequest) error
	ListDue(ctx context.Context, req *ListDueRequest) ([]Due, error)
	// Settle deletes the row, or moves it to RetryAt, only while it still
	// holds the due time that was claimed.
	Settle(ctx context.Context, req *SettleRequest) error
	// ListDueTenants lists, across every tenant, the tenants with due work
	// of the kind: the sweep's backstop for a wake that was lost.
	ListDueTenants(ctx context.Context, req *ListDueTenantsRequest) ([]pagination.TenantInfo, error)
}

type EnqueueRequest struct {
	Kind       string
	TenantInfo pagination.TenantInfo
	SubjectIDs []pulid.ID
	At         int64
}

type ListDueRequest struct {
	Kind       string
	TenantInfo pagination.TenantInfo
	Now        int64
	Limit      int
}

type Due struct {
	SubjectID pulid.ID
	DueAt     int64
}

type SettleRequest struct {
	Kind         string
	TenantInfo   pagination.TenantInfo
	SubjectID    pulid.ID
	ClaimedDueAt int64
	RetryAt      *int64
}

type ListDueTenantsRequest struct {
	Kind  string
	Now   int64
	Limit int
}
