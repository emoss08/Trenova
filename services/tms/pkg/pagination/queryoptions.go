package pagination

import "github.com/emoss08/trenova/pkg/domaintypes"

type QueryOptions struct {
	TenantInfo       TenantInfo `form:"-"`
	Pagination       Info
	Query            string                        `form:"query" json:"query"`
	FieldFilters     []domaintypes.FieldFilter     `             json:"fieldFilters"`
	FilterGroups     []domaintypes.FilterGroup     `             json:"filterGroups"`
	GeoFilters       []domaintypes.GeoFilter       `             json:"geoFilters"`
	AggregateFilters []domaintypes.AggregateFilter `             json:"aggregateFilters"`
	Sort             []domaintypes.SortField       `             json:"sort"`
}
