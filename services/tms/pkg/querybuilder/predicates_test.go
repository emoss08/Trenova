package querybuilder

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestApplyFilterPredicates_AddsNoColumnsOrOrdering(t *testing.T) {
	ClearCaches()

	db := newAdditionalTestDB()
	query := db.NewSelect().
		Model((*additionalSearchEntity)(nil)).
		ModelTableExpr("extended_search_entities AS sve").
		ColumnExpr("COUNT(*) AS total")
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Query: "alpha",
		Sort:  []domaintypes.SortField{{Field: "createdAt", Direction: dbtype.SortDirectionDesc}},
	}

	sql := ApplyFilterPredicates(query, "sve", filter, &additionalSearchEntity{}).String()

	assert.Contains(t, sql, "SELECT COUNT(*) AS total FROM")
	assert.Contains(t, sql, "websearch_to_tsquery")
	assert.Contains(t, sql, "organization_id")
	assert.NotContains(t, sql, "ts_rank")
	assert.NotContains(t, sql, "sve.*")
	assert.NotContains(t, sql, "ORDER BY")
}
