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
	// SettleMany settles a page in two statements: it deletes the subjects
	// that are done and moves the ones to retry to RetryAt, each only while
	// it still holds the due time that was claimed.
	SettleMany(ctx context.Context, req *SettleManyRequest) error
	// ListDueTenants lists, across every tenant, the tenants with due work
	// of the kind: the sweep's backstop for a wake that was lost.
	ListDueTenants(ctx context.Context, req *ListDueTenantsRequest) ([]pagination.TenantInfo, error)
	// ListDueKinds lists, across every tenant, the tenant and kind pairs with
	// due work among the kinds that start with the prefix: the backstop for a
	// family of kinds, one per agent.
	ListDueKinds(ctx context.Context, req *ListDueKindsRequest) ([]TenantKind, error)
}

type ListDueKindsRequest struct {
	Prefix string
	Now    int64
	Limit  int
}

type TenantKind struct {
	TenantInfo pagination.TenantInfo
	Kind       string
}

type EnqueueRequest struct {
	Kind       string
	TenantInfo pagination.TenantInfo
	SubjectIDs []pulid.ID
	At         int64
	// Detail is what the work needs beyond the subject; the latest request
	// replaces it.
	Detail string
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
	Detail    string
}

// SettleManyRequest is a page's outcome: Done and Retry are the claimed rows,
// each with the due time it was claimed at.
type SettleManyRequest struct {
	Kind       string
	TenantInfo pagination.TenantInfo
	Done       []Due
	Retry      []Due
	RetryAt    int64
}

type ListDueTenantsRequest struct {
	Kind  string
	Now   int64
	Limit int
}
