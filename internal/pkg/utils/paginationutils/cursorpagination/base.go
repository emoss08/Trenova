package cursorpagination

import (
	"fmt"
	"strings"

	"github.com/rotisserie/eris"
)

// BuildPaginatedQuery builds a query with cursor-based pagination
func BuildPaginatedQuery(opts *BuildPaginatedQueryOptions) error {
	// Validate and normalize page size
	if opts.PaginationOpts.PageSize <= 0 {
		opts.PaginationOpts.PageSize = DefaultPageSize
	}
	if opts.PaginationOpts.PageSize > MaxPageSize {
		opts.PaginationOpts.PageSize = MaxPageSize
	}
	// Apply cursor pagination if cursor is provided
	if opts.PaginationOpts.Cursor != nil && *opts.PaginationOpts.Cursor != "" {
		cursor, err := DecodeCursor(*opts.PaginationOpts.Cursor)
		if err != nil {
			return eris.Wrap(err, "decode cursor")
		}
		pk := opts.Model.GetPrimaryKey()
		conditions, params := buildCursorConditions(opts.Model.GetTableAlias(), pk, cursor)
		if conditions != "" {
			opts.Query.Where(conditions, params...)
		}
	}
	// Add limit for pagination
	opts.Query.Limit(opts.PaginationOpts.PageSize + 1)
	return nil
}

// buildCursorConditions creates the SQL conditions for cursor-based pagination
func buildCursorConditions(tableAlias string, pk PrimaryKey, cursor *Cursor) (string, []any) {
	var conditions []string
	var params []any
	for _, field := range pk.Fields {
		if value, ok := cursor.Values[field]; ok {
			conditions = append(conditions,
				fmt.Sprintf("%s.%s > ?", tableAlias, field))
			params = append(params, value)
		}
	}
	if len(conditions) == 0 {
		return "", nil
	}
	return fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), params
}

// ProcessResponse processes the query results into a paginated response
func ProcessResponse[T BaseModel](
	entities []T,
	total int,
	opts Query,
) (*Response[T], error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	hasMore := len(entities) > pageSize
	dataLen := len(entities)
	if hasMore {
		dataLen = pageSize
	}
	response := &Response[T]{
		Data: make([]*T, dataLen),
		Meta: Meta{
			TotalCount: total,
			HasMore:    hasMore,
		},
	}
	// Copy data with proper pointer handling
	for i := range entities[:dataLen] {
		tmp := entities[i]
		response.Data[i] = &tmp
	}

	// Handle next cursor if there are more records
	if hasMore {
		lastEntity := entities[pageSize-1]
		nextCursor, err := EncodeCursor(lastEntity.GetPrimaryKey())
		if err != nil {
			return nil, eris.Wrap(err, "encode cursor")
		}
		response.Cursors.Next = &nextCursor
	}

	// Add previous cursor if we're not on the first page
	if opts.Cursor != nil && *opts.Cursor != "" {
		if len(entities) > 0 {
			firstEntity := entities[0]
			prevCursor, err := EncodeCursor(firstEntity.GetPrimaryKey())
			if err != nil {
				return nil, eris.Wrap(err, "encode cursor")
			}
			response.Cursors.Previous = &prevCursor
		}
	}
	return response, nil
}
