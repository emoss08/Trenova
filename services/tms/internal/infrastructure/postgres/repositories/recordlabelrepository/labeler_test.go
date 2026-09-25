package recordlabelrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestLabelQueries_AreScopedToTheTenantAndNameEachRow(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ids := []pulid.ID{pulid.MustNew("rec_"), pulid.MustNew("rec_")}

	for resource, src := range sources {
		sql := labelQuery(db, tenant, &src, ids).String()

		assert.Contains(t, sql, tenant.OrgID.String(), resource)
		assert.Contains(t, sql, tenant.BuID.String(), resource)
		assert.Contains(t, sql, " AS label", resource)
		assert.Contains(t, sql, ids[0].String(), resource)
		assert.Contains(t, sql, ids[1].String(), resource)
		assert.Equal(t, 1, strings.Count(sql, " AS id"), resource)
	}
}

func TestLabelSources_CoverTheRecordsPreviewsNameMost(t *testing.T) {
	t.Parallel()

	for _, resource := range []permission.Resource{
		permission.ResourceShipment,
		permission.ResourceCustomer,
		permission.ResourceLocation,
		permission.ResourceWorker,
		permission.ResourceTractor,
		permission.ResourceTrailer,
		permission.ResourceCarrier,
		permission.ResourceInvoice,
		permission.ResourceDocument,
	} {
		_, ok := sources[resource]
		assert.True(t, ok, resource)
	}
}

func TestDistinctIDs_DropsBlanksAndDuplicatesWithinTheBound(t *testing.T) {
	t.Parallel()

	a, b := pulid.MustNew("rec_"), pulid.MustNew("rec_")
	assert.Equal(t, []pulid.ID{a, b}, distinctIDs([]pulid.ID{a, "", b, a}))

	many := make([]pulid.ID, 0, MaxIDsPerResource+10)
	for range MaxIDsPerResource + 10 {
		many = append(many, pulid.MustNew("rec_"))
	}
	assert.Len(t, distinctIDs(many), MaxIDsPerResource)
}
