// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package handlers

import "github.com/emoss08/trenova/internal/pkg/utils/paginationutils/cursorpagination"

type BaseHandlerRequest struct {
	QueryOpts cursorpagination.Query
}
