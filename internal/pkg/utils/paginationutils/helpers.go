package paginationutils

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/gofiber/fiber/v2"
)

// ParseEnhancedQueryFromJSON parses enhanced query parameters from JSON strings
// This is a helper for handlers that are manually parsing JSON filters and sorts
func ParseEnhancedQueryFromJSON(
	c *fiber.Ctx,
	reqCtx *appctx.RequestContext,
) (*ports.QueryOptions, error) {
	enhancedOpts := &ports.QueryOptions{
		Limit:  c.QueryInt("limit", 10),
		Offset: c.QueryInt("offset", 0),
		Query:  c.Query("query"),
		TenantOpts: &ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},

		FieldFilters: []ports.FieldFilter{},
		Sort:         []ports.SortField{},
	}

	// Parse filters from JSON string
	if filtersStr := c.Query("filters"); filtersStr != "" {
		var filters []ports.FieldFilter
		if err := sonic.Unmarshal([]byte(filtersStr), &filters); err != nil {
			return nil, fiber.NewError(
				fiber.StatusBadRequest,
				"Invalid filters format: "+err.Error(),
			)
		}
		enhancedOpts.FieldFilters = filters
	}

	// Parse sort from JSON string
	if sortStr := c.Query("sort"); sortStr != "" {
		var sorts []ports.SortField
		if err := sonic.Unmarshal([]byte(sortStr), &sorts); err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid sort format: "+err.Error())
		}
		enhancedOpts.Sort = sorts
	}

	return enhancedOpts, nil
}

// ParseAdditionalQueryParams is a generic helper to parse additional boolean query parameters
func ParseAdditionalQueryParams[T any](c *fiber.Ctx, params *T) error {
	return c.QueryParser(params)
}

// BuildListOptionsWithFilter creates a standardized list options structure
// This helps consolidate the creation of list options across different handlers
func BuildListOptionsWithFilter[T any](
	enhancedOpts *ports.QueryOptions,
	additionalOpts T,
) any {
	// This is a generic builder that handlers can use as a template
	// Each handler will need to create their specific ListOptions type
	return struct {
		Filter         *ports.LimitOffsetQueryOptions
		EnhancedFilter *ports.QueryOptions
		Additional     T
	}{
		Filter: &ports.LimitOffsetQueryOptions{
			Limit:      enhancedOpts.Limit,
			Offset:     enhancedOpts.Offset,
			Query:      enhancedOpts.Query,
			TenantOpts: enhancedOpts.TenantOpts,
		},
		EnhancedFilter: enhancedOpts,
		Additional:     additionalOpts,
	}
}

// MigrateToEnhancedPagination provides a migration path from old pagination to enhanced
// This wrapper allows handlers to gradually migrate without breaking existing functionality
func MigrateToEnhancedPagination[T any](
	c *fiber.Ctx,
	reqCtx *appctx.RequestContext,
	fieldConfig *ports.FieldConfiguration,
	handler func(*fiber.Ctx, *ports.QueryOptions) (*ports.ListResult[T], error),
) (*ports.QueryOptions, *ports.ListResult[T], error) {
	// Try to parse using enhanced method first (array format)
	opts, err := limitoffsetpagination.ParseEnhancedParams(c, reqCtx, fieldConfig)
	if err == nil && (opts.HasFilters() || opts.HasSort()) {
		// Enhanced format detected, use it
		result, err := handler(c, opts)
		return opts, result, err
	}

	// Fall back to JSON string format (for backward compatibility)
	opts, err = ParseEnhancedQueryFromJSON(c, reqCtx)
	if err != nil {
		return nil, nil, err
	}

	// Validate if field config is provided
	if fieldConfig != nil {
		if err := opts.ValidateFilters(fieldConfig.FilterableFields); err != nil {
			return nil, nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if err := opts.ValidateSort(fieldConfig.SortableFields); err != nil {
			return nil, nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	result, err := handler(c, opts)
	return opts, result, err
}
