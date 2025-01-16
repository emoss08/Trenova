package ports

import (
	"github.com/gofiber/fiber/v2"
	"github.com/trenova-app/transport/internal/pkg/utils/paginationutils/cursorpagination"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type SortOrder string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

// TenantOptions is a struct that contains the options for a tenant
type TenantOptions struct {
	BuID   pulid.ID `json:"buId"`
	OrgID  pulid.ID `json:"orgId"`
	UserID pulid.ID `json:"userId"`
}

// FilterQueryOptions is a struct that contains the options for a filter query
type FilterQueryOptions struct {
	// ID of the business unit
	BuID pulid.ID

	// ID of the organization
	OrgID pulid.ID

	// ID of the user making the request
	UserID pulid.ID

	// Pagination options
	PaginationOpts cursorpagination.Query

	// Query string
	Query string
}

// LimitOffsetQueryOptions is a struct that contains the options for a limit/offset pagination
type LimitOffsetQueryOptions struct {
	TenantOpts *TenantOptions `json:"tenantOpts"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	Query      string         `json:"query" query:"search"`
}

type Response[T any] struct {
	Results T      `json:"results"`
	Count   int    `json:"count"`
	Next    string `json:"next"`
	Prev    string `json:"previous"`
}

// ListResult is a struct that contains the items and total count of a list
type ListResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

// PageableHandler is a function that handles a pageable request
type PageableHandler[T any] func(ctx *fiber.Ctx, opts *LimitOffsetQueryOptions) (*ListResult[T], error)
