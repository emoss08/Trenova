package handlers

import "github.com/emoss08/trenova/internal/pkg/utils/paginationutils/cursorpagination"

type BaseHandlerRequest struct {
	QueryOpts cursorpagination.Query
}
