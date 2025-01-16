package handlers

import "github.com/trenova-app/transport/internal/pkg/utils/paginationutils/cursorpagination"

type BaseHandlerRequest struct {
	QueryOpts cursorpagination.Query
}
