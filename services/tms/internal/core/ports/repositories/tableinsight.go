package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

type TableInsightScope struct {
	Filter  *pagination.QueryOptions
	Options map[string]any
}

type TableFacetRequest struct {
	Scope TableInsightScope
	Field string
	Limit int
}

type TableFacetBucket struct {
	Value *string
	Count int
}

type TableFacetResult struct {
	Buckets []TableFacetBucket
	Total   int
}

type TableAggregateRequest struct {
	Scope  TableInsightScope
	Fields []string
}

type TableAggregateValue struct {
	Field   string
	Sum     *decimal.Decimal
	Average *decimal.Decimal
	Min     *decimal.Decimal
	Max     *decimal.Decimal
}

type TableAggregateResult struct {
	Count  int
	Values []TableAggregateValue
}

type TableMatchRequest struct {
	Scope TableInsightScope
	Limit int
}

type TableMatchResult struct {
	IDs   []string
	Total int
}

type TableSeriesRequest struct {
	Scope       TableInsightScope
	GroupField  string
	GroupValues []string
	DateField   string
	ValueField  string
	Bounds      []int64
}

type TableSeriesGroup struct {
	Value  string
	Points []decimal.Decimal
}

type TableSeriesResult struct {
	Groups []TableSeriesGroup
}

type TableInsightField struct {
	Name      string
	Facetable bool
	Summable  bool
	Timeline  bool
}

type TableInsightSource interface {
	Resource() permission.Resource
	InsightFields() []TableInsightField
	Facet(ctx context.Context, req *TableFacetRequest) (*TableFacetResult, error)
	Aggregate(ctx context.Context, req *TableAggregateRequest) (*TableAggregateResult, error)
	MatchingIDs(ctx context.Context, req *TableMatchRequest) (*TableMatchResult, error)
	Series(ctx context.Context, req *TableSeriesRequest) (*TableSeriesResult, error)
}
