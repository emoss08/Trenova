/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package limitoffsetpagination

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gofiber/fiber/v2"
)

func HandlePaginatedRequest[T any](
	c *fiber.Ctx,
	eh *validator.ErrorHandler,
	reqCtx *appctx.RequestContext,
	handler ports.PageableHandler[T],
) error {
	pg, err := Params(c)
	if err != nil {
		return eh.HandleError(c, err)
	}

	filter := &ports.LimitOffsetQueryOptions{
		TenantOpts: ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},
		Limit:  pg.Limit,
		Offset: pg.Offset,
	}

	result, err := handler(c, filter)
	if err != nil {
		return eh.HandleError(c, err)
	}

	nextURL := GetNextPageURL(c, pg.Limit, pg.Offset, result.Total)
	prevURL := GetPrevPageURL(c, pg.Limit, pg.Offset)

	return c.JSON(ports.Response[[]T]{
		Count:   result.Total,
		Results: result.Items,
		Next:    nextURL,
		Prev:    prevURL,
	})
}

func HandleEnhancedPaginatedRequest[T any](
	c *fiber.Ctx,
	eh *validator.ErrorHandler,
	reqCtx *appctx.RequestContext,
	handler ports.EnhancedPageableHandler[T],
) error {
	pg, err := Params(c)
	if err != nil {
		return eh.HandleError(c, err)
	}

	filter := &ports.QueryOptions{
		TenantOpts: ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},
		Limit:  pg.Limit,
		Offset: pg.Offset,
	}

	result, err := handler(c, filter)
	if err != nil {
		return eh.HandleError(c, err)
	}

	nextURL := GetNextPageURL(c, pg.Limit, pg.Offset, result.Total)
	prevURL := GetPrevPageURL(c, pg.Limit, pg.Offset)

	return c.JSON(ports.Response[[]T]{
		Count:   result.Total,
		Results: result.Items,
		Next:    nextURL,
		Prev:    prevURL,
	})
}

func Params(c *fiber.Ctx) (*Info, error) {
	// Default values
	defaultOffset := 0
	defaultLimit := 10

	offsetStr := c.Query("offset")
	limitStr := c.Query("limit")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = defaultOffset
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = defaultLimit
	}

	return &Info{
		Offset: offset,
		Limit:  limit,
	}, nil
}

func buildPageURL(c *fiber.Ctx, offset, limit int) string {
	query := c.Request().URI().QueryArgs()
	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(limit))

	scheme := "http"
	if c.Secure() {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s?%s", scheme, c.Hostname(), c.Path(), query.QueryString())
}

func GetNextPageURL(c *fiber.Ctx, limit, offset, totalRows int) string {
	if offset+limit >= totalRows {
		return ""
	}
	return buildPageURL(c, offset+limit, limit)
}

func GetPrevPageURL(c *fiber.Ctx, limit, offset int) string {
	if offset == 0 {
		return ""
	}
	prevOffset := max(offset-limit, 0)
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return buildPageURL(c, prevOffset, limit)
}

// ParseEnhancedParams parses the enhanced query parameters from the request
//
// Parameters:
//   - c: The fiber context
//   - reqCtx: The request context
//   - fieldConfig: The field configuration
//
// Returns:
//   - *ports.EnhancedQueryOptions: The enhanced query options
func ParseEnhancedParams(
	c *fiber.Ctx,
	reqCtx *appctx.RequestContext,
	fieldConfig *ports.FieldConfiguration,
) (*ports.QueryOptions, error) {
	// * Parse basic pagination parameters
	pg, err := Params(c)
	if err != nil {
		return nil, err
	}

	opts := &ports.QueryOptions{
		Limit:  pg.Limit,
		Offset: pg.Offset,
		Query:  c.Query("query", ""),
		TenantOpts: ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},
	}

	// * Parse ID parameter if present
	if idStr := c.Query("id"); idStr != "" {
		// * Validate PULID format before assignment
		if id, idErr := pulid.Parse(idStr); idErr == nil {
			opts.ID = &id
		}
		// ! Silently ignore invalid PULID format
	}

	parseFilters(c, opts)
	parseSort(c, opts)

	// * Validate filters and sorting against allowed fields
	if fieldConfig != nil {
		if err = opts.ValidateFilters(fieldConfig.FilterableFields); err != nil {
			return nil, err
		}
		if err = opts.ValidateSort(fieldConfig.SortableFields); err != nil {
			return nil, err
		}
	}

	return opts, nil
}

// parseFilters parses filter parameters from the request
func parseFilters(c *fiber.Ctx, opts *ports.QueryOptions) {
	filtersMap := make(map[int]map[string]string)

	// * Parse all filter-related query parameters
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if strings.HasPrefix(keyStr, "filters[") {
			// * Extract index and field name
			// * filters[0][field] -> index: 0, field: field
			if parts := parseFilterKey(keyStr); len(parts) == 2 {
				index, _ := strconv.Atoi(parts[0])
				field := parts[1]

				if filtersMap[index] == nil {
					filtersMap[index] = make(map[string]string)
				}
				filtersMap[index][field] = string(value)
			}
		}
	})

	// * Convert map to filter structs
	for _, filterMap := range filtersMap {
		if field, hasField := filterMap["field"]; hasField {
			if operator, hasOperator := filterMap["operator"]; hasOperator {
				if value, hasValue := filterMap["value"]; hasValue {
					filter := ports.FieldFilter{
						Field:    field,
						Operator: ports.FilterOperator(operator),
						Value:    parseFilterValue(value, operator),
					}
					opts.FieldFilters = append(opts.FieldFilters, filter)
				}
			}
		}
	}
}

// parseSort parses sort parameters from the request
// expected format: sort[0][field]=created_at&sort[0][direction]=desc
func parseSort(c *fiber.Ctx, opts *ports.QueryOptions) {
	sortMap := make(map[int]map[string]string)

	// * Parse all sort-related query parameters
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if strings.HasPrefix(keyStr, "sort[") {
			// * Extract index and field name
			if parts := parseFilterKey(keyStr); len(parts) == 2 {
				index, _ := strconv.Atoi(parts[0])
				field := parts[1]

				if sortMap[index] == nil {
					sortMap[index] = make(map[string]string)
				}
				sortMap[index][field] = string(value)
			}
		}
	})

	// Convert map to sort structs
	for _, sortEntry := range sortMap {
		if field, hasField := sortEntry["field"]; hasField {
			direction := sortEntry["direction"]
			if direction == "" {
				direction = "asc"
			}

			sort := ports.SortField{
				Field:     field,
				Direction: ports.SortDirection(direction),
			}
			opts.Sort = append(opts.Sort, sort)
		}
	}
}

// parseFilterKey parses a filter key like "filters[0][field]" to extract index and field
func parseFilterKey(key string) []string {
	// * Remove prefix and suffix to get "0][field"
	if strings.HasPrefix(key, "filters[") {
		key = key[8:] // Remove "filters["
	} else if strings.HasPrefix(key, "sort[") {
		key = key[5:] // Remove "sort["
	}

	// * Find the first ']' to separate index from field
	if idx := strings.Index(key, "]["); idx != -1 {
		index := key[:idx]
		field := strings.TrimSuffix(key[idx+2:], "]")
		return []string{index, field}
	}

	return nil
}

// parseFilterValue parses and converts filter values based on the operator
func parseFilterValue(value, operator string) any {
	switch ports.FilterOperator(operator) { //nolint:exhaustive // We only support the operators we need
	case ports.OpIn, ports.OpNotIn:
		// * For 'in' operators, try to parse as JSON array
		var arr []any
		if err := sonic.Unmarshal([]byte(value), &arr); err == nil {
			return arr
		}
		// * If JSON parsing fails, split by comma
		return strings.Split(value, ",")
	case ports.OpDateRange:
		// * For date range, try to parse as JSON object
		var dateRange map[string]any
		if err := sonic.Unmarshal([]byte(value), &dateRange); err == nil {
			return dateRange
		}
		return value
	default:
		return value
	}
}
