package gqlconnection

import "github.com/emoss08/trenova/pkg/pagination"

// RecordListLimit is how many rows a plain list keyed by an optional record
// returns: what the caller asked for, within the page bounds; when it asked
// for none, every row of the one record it names, or one default page when
// it names none and spans the organization.
func RecordListLimit(limit *int, scoped bool) int {
	if limit != nil {
		return pagination.ClampLimit(*limit)
	}
	if scoped {
		return 0
	}

	return pagination.DefaultLimit
}
