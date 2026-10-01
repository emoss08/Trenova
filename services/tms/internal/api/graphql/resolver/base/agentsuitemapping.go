package base

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
)

func PageInfoFor(hasNext bool, lastCursor string) *gqlmodel.PageInfo {
	info := &gqlmodel.PageInfo{HasNextPage: hasNext}
	if lastCursor != "" {
		cursor := lastCursor
		info.EndCursor = &cursor
	}

	return info
}
