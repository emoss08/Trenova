package formulatemplaterepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestStatsQueryCountsEveryConsumerInOneStatement(t *testing.T) {
	t.Parallel()

	ids := []pulid.ID{pulid.MustNew("ft_"), pulid.MustNew("ft_")}
	sql, args := keeltest.Render(t, testRepository().statsQuery(ids))

	for _, table := range []string{`"shipments" s`, `"rate_agreement_rules"`} {
		assert.Contains(t, sql, table, "every consumer of a template is counted")
	}
	assert.Contains(t, sql, `s."organization_id" = $`)
	assert.Contains(t, sql, `s."formula_template_id" = ft."id"`)
	assert.Contains(t, sql, "LIMIT 2")
	assert.Contains(t, args, keeltest.TestOrganizationID.String())
	assert.Equal(t, 1, strings.Count(sql, `FROM "formula_templates" ft`),
		"the page is one statement over formula_templates")
}
