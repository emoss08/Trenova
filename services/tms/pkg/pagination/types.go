package pagination

import "github.com/emoss08/trenova/shared/pulid"

type TenantInfo struct {
	OrgID  pulid.ID `json:"orgId"`
	BuID   pulid.ID `json:"buId"`
	UserID pulid.ID `json:"userId"`
}

type ListResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type SelectQueryRequest struct {
	TenantInfo TenantInfo
	Pagination Info
	Query      string `json:"query"`
}

type Info struct {
	Limit  int `json:"limit"  default:"20" form:"limit"  binding:"min=1,max=100"`
	Offset int `json:"offset" default:"0"  form:"offset" binding:"min=0"`
}

type Response[T any] struct {
	Results T      `json:"results"`
	Count   int    `json:"count"`
	Next    string `json:"next"`
	Prev    string `json:"previous"`
}
