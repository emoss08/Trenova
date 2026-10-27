package commodityrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCommoditiesByIDs_JoinsTheHazardousMaterialUnderTheTenant(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, commoditiesByIDs([]pulid.ID{"com_1", "com_2"}))

	assert.Contains(t, sql, `FROM "commodities" c LEFT JOIN "hazardous_materials" hm ON `)
	assert.Contains(t, sql, `hm."id" = c."hazardous_material_id"`)
	assert.Contains(t, sql, `c."id" = ANY($`)
	assert.Contains(t, sql, `LIMIT 2`)
	assert.Contains(t, args, []string{"com_1", "com_2"})
}
