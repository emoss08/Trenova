package pagination

import "github.com/emoss08/trenova/shared/pulid"

type TenantInfo struct {
	OrgID  pulid.ID `json:"orgId"`
	BuID   pulid.ID `json:"buId"`
	UserID pulid.ID `json:"userId"`
}

type ListResult[T any] struct {
	Items       []T               `json:"items"`
	Total       int               `json:"total"`
	HasNextPage bool              `json:"hasNextPage,omitempty"`
	CursorSort  []CursorSortField `json:"-"`
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

func ClampLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultLimit
	case limit > MaxLimit:
		return MaxLimit
	default:
		return limit
	}
}

func ClampOffset(offset int) int {
	if offset < 0 {
		return DefaultOffset
	}

	return offset
}

func (i Info) SafeLimit() int {
	if i.Limit == MaxLimit+1 {
		return i.Limit
	}

	return ClampLimit(i.Limit)
}

func (i Info) SafeOffset() int {
	return ClampOffset(i.Offset)
}

type Response[T any] struct {
	Results T      `json:"results"`
	Count   int    `json:"count"`
	Next    string `json:"next"`
	Prev    string `json:"previous"`
}

type CursorResponse[T any] struct {
	Results     T      `json:"results"`
	Count       int    `json:"count"`
	TotalCount  *int   `json:"totalCount"`
	Next        string `json:"next"`
	Previous    string `json:"previous"`
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}
