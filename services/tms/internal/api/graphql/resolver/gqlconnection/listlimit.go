package gqlconnection

import "github.com/emoss08/trenova/pkg/pagination"

// RecordListLimit is how many rows a plain list keyed by an optional record
// returns: what the caller asked for, within the page bounds, one full page
// for a named record, and one default page across the organization. Only a
// list an agent tool reads whole stays a plain list; every other list of a
// record's rows is a connection.
func RecordListLimit(limit *int, scoped bool) int {
	if limit != nil {
		return pagination.ClampLimit(*limit)
	}
	if scoped {
		return pagination.MaxLimit
	}

	return pagination.DefaultLimit
}
