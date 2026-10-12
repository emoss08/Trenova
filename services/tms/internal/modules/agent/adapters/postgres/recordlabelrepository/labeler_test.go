package recordlabelrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
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
		permission.ResourceDriverSettlement,
		permission.ResourceCarrierSettlement,
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

// A person unticks shipments from every one a transfer proposed, each named by
// its label, so one read names every record the longest subset may offer.
func TestDistinctIDs_KeepsEveryRecordASubsetMayOffer(t *testing.T) {
	t.Parallel()

	offered := make([]pulid.ID, 0, toolschema.MaxSubsetChoices)
	for range toolschema.MaxSubsetChoices {
		offered = append(offered, pulid.MustNew("shp_"))
	}

	assert.Len(t, distinctIDs(offered), toolschema.MaxSubsetChoices)
}

func TestOwnedLabelQueries_AreScopedToTheTenantAndCarryTheOwner(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ids := []pulid.ID{pulid.MustNew("rec_"), pulid.MustNew("rec_")}

	for resource, src := range ownedSources {
		_, named := sources[src.owner]
		assert.True(t, named, "%s is named by %s, which has no label", resource, src.owner)

		sql := ownedQuery(db, tenant, &src, ids).String()

		assert.Contains(t, sql, tenant.OrgID.String(), resource)
		assert.Contains(t, sql, tenant.BuID.String(), resource)
		assert.Contains(t, sql, " AS owner_id", resource)
		assert.Contains(t, sql, " AS label", resource)
		assert.Contains(t, sql, ids[0].String(), resource)
		assert.Contains(t, sql, ids[1].String(), resource)
	}
}

func TestUserLabelQuery_NamesOnlyPeopleOfTheTenant(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	id := pulid.MustNew("usr_")

	sql := userQuery(db, tenant, []pulid.ID{id}).String()

	assert.Contains(t, sql, id.String())
	assert.Contains(t, sql, "user_organization_memberships")
	assert.Contains(t, sql, "EXISTS")
	assert.Equal(t, 2, strings.Count(sql, tenant.OrgID.String()), sql)
	assert.Equal(t, 2, strings.Count(sql, tenant.BuID.String()), sql)
	assert.NotContains(t, sql, "password")
	assert.NotContains(t, sql, "email_address")
}

func TestJoinLabel_NamesTheOwnerThenWhatTheRecordIsOfIt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Jane Doe · Medical", joinLabel("Jane Doe", "Medical"))
	assert.Equal(t, "Jane Doe", joinLabel(" Jane Doe ", ""))
	assert.Equal(t, "Medical", joinLabel("", "Medical"))
	assert.Empty(t, joinLabel(" ", " "))
}
