package hazmatsegregationrulerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestActiveRules_ReadsTheTenantsActiveRulesByName(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, activeRules())

	assert.Contains(t, sql, `FROM "hazmat_segregation_rules" hsr WHERE `)
	assert.Contains(t, sql, `hsr."status" = 'Active'`)
	assert.Contains(t, sql, `ORDER BY hsr."name" ASC, hsr."id" ASC`)
	assert.Equal(t, []any{"bu_1", "org_1"}, args)
}
