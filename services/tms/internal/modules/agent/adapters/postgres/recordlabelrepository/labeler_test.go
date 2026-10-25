package recordlabelrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func idArg(args []any, ids []pulid.ID) bool {
	for _, arg := range args {
		values, ok := arg.([]string)
		if !ok || len(values) != len(ids) {
			continue
		}
		match := true
		for i, id := range ids {
			if values[i] != id.String() {
				match = false
			}
		}
		if match {
			return true
		}
	}

	return false
}

func TestLabelQueries_AreScopedToTheTenantAndNameEachRow(t *testing.T) {
	t.Parallel()

	ids := []pulid.ID{pulid.MustNew("rec_"), pulid.MustNew("rec_")}

	for resource, src := range sources {
		sql, args := keeltest.Render(t, src(ids))

		assert.Contains(t, args, keeltest.TestOrganizationID.String(), resource)
		assert.Contains(t, args, keeltest.TestBusinessUnitID.String(), resource)
		assert.Contains(t, sql, "organization_id", resource)
		assert.Contains(t, sql, "business_unit_id", resource)
		assert.Contains(t, sql, "COALESCE((", resource)
		assert.True(t, idArg(args, ids), "%s: %v", resource, args)
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

	ids := []pulid.ID{pulid.MustNew("rec_"), pulid.MustNew("rec_")}

	for resource, src := range ownedSources {
		_, named := sources[src.owner]
		assert.True(t, named, "%s is named by %s, which has no label", resource, src.owner)

		sql, args := keeltest.Render(t, src.query(ids))

		assert.Contains(t, args, keeltest.TestOrganizationID.String(), resource)
		assert.Contains(t, args, keeltest.TestBusinessUnitID.String(), resource)
		assert.Contains(t, sql, "organization_id", resource)
		assert.True(t, idArg(args, ids), "%s: %v", resource, args)
	}
}

func TestUserLabelQuery_NamesOnlyPeopleOfTheTenant(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{
		OrgID: keeltest.TestOrganizationID,
		BuID:  keeltest.TestBusinessUnitID,
	}
	id := pulid.MustNew("usr_")

	sql, args := keeltest.Render(t, userQuery(tenant, []pulid.ID{id}))

	assert.True(t, idArg(args, []pulid.ID{id}), args)
	assert.Contains(t, sql, "user_organization_memberships")
	assert.Contains(t, sql, "EXISTS")
	assert.Contains(t, sql, "current_organization_id")
	assert.Contains(t, args, keeltest.TestOrganizationID.String())
	assert.Contains(t, args, keeltest.TestBusinessUnitID.String())
	assert.NotContains(t, sql, "password")
	assert.NotContains(t, sql, "email_address")
}

func TestMovesQuery_ReadsTheMovesOfTheTenant(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("smv_")
	sql, args := keeltest.Render(t, movesQuery([]pulid.ID{id}))

	assert.Contains(t, sql, "shipment_moves")
	assert.Contains(t, sql, "sequence")
	assert.True(t, idArg(args, []pulid.ID{id}), args)
	assert.Contains(t, args, keeltest.TestOrganizationID.String())
}

func TestJoinLabel_NamesTheOwnerThenWhatTheRecordIsOfIt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Jane Doe · Medical", joinLabel("Jane Doe", "Medical"))
	assert.Equal(t, "Jane Doe", joinLabel(" Jane Doe ", ""))
	assert.Equal(t, "Medical", joinLabel("", "Medical"))
	assert.Empty(t, joinLabel(" ", " "))
}
