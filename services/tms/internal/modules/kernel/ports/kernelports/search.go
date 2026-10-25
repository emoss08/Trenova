package kernelports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
)

type GlobalSearchRequest struct {
	Query       string
	TenantInfo  pagination.TenantInfo
	Principal   PrincipalInfo
	Limit       int
	EntityTypes []search.EntityType
}

type PrincipalInfo struct {
	Type     PrincipalType
	ID       pulid.ID
	UserID   pulid.ID
	APIKeyID pulid.ID
}

type GlobalSearchResult struct {
	Query  string               `json:"query"`
	Groups []*GlobalSearchGroup `json:"groups"`
}

type GlobalSearchGroup struct {
	EntityType search.EntityType  `json:"entityType"`
	Label      string             `json:"label"`
	Total      int                `json:"total"`
	Hits       []*GlobalSearchHit `json:"hits"`
}

type GlobalSearchHit struct {
	ID         string            `json:"id"`
	EntityType search.EntityType `json:"entityType"`
	Title      string            `json:"title"`
	Subtitle   string            `json:"subtitle,omitempty"`
	Href       string            `json:"href"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type GlobalSearchService interface {
	Search(ctx context.Context, req *GlobalSearchRequest) (*GlobalSearchResult, error)
}

type ListCustomerOpenBalancesRequest struct {
	TenantInfo  pagination.TenantInfo
	CustomerIDs []pulid.ID
}

type CustomerOpenBalance struct {
	CustomerID   pulid.ID
	OpenMinor    int64
	OverdueMinor int64
	CurrencyCode string
}

// CustomerOpenBalanceReader is provided by billing so a search can show what a
// customer owes beside its name without the search depending on billing.
type CustomerOpenBalanceReader interface {
	ListCustomerOpenBalances(
		ctx context.Context,
		req *ListCustomerOpenBalancesRequest,
	) ([]*CustomerOpenBalance, error)
}

type ListVisibleRecordsRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       string
	IDs        []pulid.ID
}

// RecordVisibility answers, through the database's read scope, which of a
// set of records the current request may read. A search index cannot apply
// an own or team scope, so its hits for a scoped kind go through it. It does
// not check the kind's read: the database shows every row of a kind the
// person cannot read (a record linking to one shows its reference view), so
// a caller refuses such a kind before asking.
type RecordVisibility interface {
	Scoped(kind string) bool
	VisibleIDs(ctx context.Context, req *ListVisibleRecordsRequest) ([]pulid.ID, error)
}
