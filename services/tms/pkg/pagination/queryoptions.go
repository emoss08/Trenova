package pagination

import "github.com/emoss08/trenova/pkg/domaintypes"

type QueryOptions struct {
	TenantInfo       TenantInfo
	Pagination       Info
	Query            string                        `json:"query"            form:"query"`
	FieldFilters     []domaintypes.FieldFilter     `json:"fieldFilters"`
	FilterGroups     []domaintypes.FilterGroup     `json:"filterGroups"`
	GeoFilters       []domaintypes.GeoFilter       `json:"geoFilters"`
	AggregateFilters []domaintypes.AggregateFilter `json:"aggregateFilters"`
	Sort             []domaintypes.SortField       `json:"sort"`
}
