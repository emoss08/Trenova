//go:build integration

package agentread_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentmanufacturerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededTenant struct {
	actor   *permissionports.RequestActor
	active  []pulid.ID
	retired pulid.ID
}

// seedTenant writes one tenant with active manufacturers and one inactive
// one, straight through the admin connection, which row-level security does
// not hold.
func seedTenant(t *testing.T, admin *keel.DB, label string, active int) *seededTenant {
	t.Helper()

	written := agentreadtest.SeedTenant(t, admin, label)
	tenant := &seededTenant{actor: written.Actor}
	insert := func(name, status string, createdAt int64) pulid.ID {
		t.Helper()
		id := pulid.MustNew("em_")
		_, err := keel.Exec(t.Context(), admin, `INSERT INTO equipment_manufacturers (id,
			business_unit_id, organization_id, status, name, description, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, '', $6, $7)`,
			id.String(), tenant.actor.BusinessUnitID.String(), tenant.actor.OrganizationID.String(),
			status, name, createdAt, createdAt)
		require.NoError(t, err)
		return id
	}
	for i := range active {
		tenant.active = append(tenant.active,
			insert(fmt.Sprintf("%s Freightliner %d", label, i), "Active", int64(1_760_000_000+i)))
	}
	tenant.retired = insert(label+" Retired Works", "Inactive", 1_750_000_000)

	return tenant
}

func TestReader_PagesAndFiltersARealConnectionUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	mine := seedTenant(t, admin, "A", 5)
	theirs := seedTenant(t, admin, "B", 3)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	reader := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &mocks.DenyResourcesPermissionEngine{},
		Resolvers: agentread.ResolverParams{
			EquipmentManufacturerService: equipmentmanufacturerservice.New(
				equipmentmanufacturerservice.Params{
					Logger: zap.NewNop(),
					Repo: equipmentmanufacturerrepository.New(equipmentmanufacturerrepository.Params{
						DB: conn, Logger: zap.NewNop(),
					}),
				},
			),
		},
	})
	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: mine.actor.OrganizationID,
		BusinessUnitID: mine.actor.BusinessUnitID,
		UserID:         mine.actor.UserID,
	})
	read := func(t *testing.T, input map[string]any) map[string]any {
		t.Helper()
		result, err := agentreadtest.Read(ctx, reader, mine.actor, manufacturerTable,
			map[string]any{"includeTotalCount": false, "input": input}, permission.SensitivityInternal)
		require.NoError(t, err)
		return connectionOf(t, result)
	}

	t.Run("two pages cover the tenant's active rows once and nothing of another's", func(t *testing.T) {
		active := []any{map[string]any{"field": "status", "operator": "in", "value": []any{"Active"}}}
		first := read(t, map[string]any{"first": 3, "fieldFilters": active})
		info := first["pageInfo"].(map[string]any)
		require.Equal(t, true, info["hasNextPage"])
		second := read(t, map[string]any{"first": 3, "after": info["endCursor"], "fieldFilters": active})
		assert.Equal(t, false, second["pageInfo"].(map[string]any)["hasNextPage"])

		seen := append(nodeIDs(t, first), nodeIDs(t, second)...)
		want := make([]string, 0, len(mine.active))
		for _, id := range mine.active {
			want = append(want, id.String())
		}
		assert.ElementsMatch(t, want, seen, "every active row once, and no other")
		for _, id := range theirs.active {
			assert.NotContains(t, seen, id.String())
		}
	})

	t.Run("a typed status filter narrows the connection", func(t *testing.T) {
		inactive := read(t, map[string]any{"first": 10, "fieldFilters": []any{
			map[string]any{"field": "status", "operator": "in", "value": []any{"Inactive"}},
		}})

		assert.Equal(t, []string{mine.retired.String()}, nodeIDs(t, inactive))
	})
}
