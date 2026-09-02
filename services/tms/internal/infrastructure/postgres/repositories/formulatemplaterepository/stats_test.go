package formulatemplaterepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// TestStatsQueryCountsEveryConsumerInOneStatement guards the list page's
// per-row counts: one statement for the whole page, every table that can
// reference a template counted, tenant-scoped on every subselect.
func TestStatsQueryCountsEveryConsumerInOneStatement(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ids := []pulid.ID{pulid.MustNew("ft_"), pulid.MustNew("ft_")}

	query := statsQuery(db.NewSelect(), &repositories.GetFormulaTemplateStatsRequest{
		TenantInfo:  tenant,
		TemplateIDs: ids,
	})

	sqlText, err := query.AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)
	text := string(sqlText)

	for _, table := range []string{
		"shipments",
		"accessorial_charges",
		"rate_agreement_rules",
		"rate_agreement_accessorials",
		"formula_template_test_cases",
	} {
		assert.Contains(t, text, table, "every consumer of a template is counted")
	}
	assert.Contains(t, text, "usage_count")
	assert.Contains(t, text, "scenario_count")
	assert.Contains(t, text, tenant.OrgID.String())
	assert.Contains(t, text, ids[0].String())
	assert.Contains(t, text, ids[1].String())
	assert.Equal(t, 1, countOccurrences(text, "FROM formula_templates AS ft"),
		"the page is one statement over formula_templates")
}

func countOccurrences(text, needle string) int {
	count := 0
	for i := 0; ; {
		j := indexFrom(text, needle, i)
		if j < 0 {
			return count
		}
		count++
		i = j + len(needle)
	}
}

func indexFrom(text, needle string, from int) int {
	for i := from; i+len(needle) <= len(text); i++ {
		if text[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
