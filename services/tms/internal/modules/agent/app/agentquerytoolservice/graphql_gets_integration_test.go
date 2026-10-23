//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carrierintelrepository"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// convertedGets names each converted get by the converted list whose rows
// it reads one of.
var convertedGets = map[string]string{
	"customer_list":           "customer_get",
	"location_list":           "location_get",
	"carrier_list":            "carrier_get",
	"equipment_list_tractors": "equipment_get_tractor",
	"equipment_list_trailers": "equipment_get_trailer",
}

func TestConvertedGets_ReadOneRecordOfTheTenantUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	lists, services := convertedListCases(conn)
	events := carrierintelrepository.NewEventRepository(carrierintelrepository.Params{
		DB: conn, Logger: zap.NewNop(),
	})
	services.CarrierIntelService = carrierintelservice.New(carrierintelservice.Params{
		Logger: zap.NewNop(), Config: &config.Config{}, EventRepo: events,
	})

	gets := make(map[string]*referenceCase, len(convertedGets)+1)
	for idx := range lists {
		if name, ok := convertedGets[lists[idx].tool]; ok {
			gets[name] = &lists[idx]
		}
	}
	gets["carrier_get_intelligence_event"] = &referenceCase{
		tool: "carrier_get_intelligence_event",
		write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, _ bool) (pulid.ID, error) {
			bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
			rows, err := events.InsertIgnoreDuplicates(ctx, []*carrierintel.CarrierIntelEvent{{
				ID: pulid.MustNew("cievt_"), BusinessUnitID: bu, OrganizationID: org,
				SubjectType: carrierintel.SubjectTypeCarrier, SubjectID: "DOT" + label,
				DOTNumber: "100" + label[1:], SubjectName: "Carrier " + label,
				Provider: integration.TypeCarrierOK, Source: carrierintel.EventSourceSnapshotDiff,
				Category: carrierintel.SectionInsurance, Severity: carrierintel.SeverityHigh,
				PriorValue: "Active", CurrentValue: "Lapsed", Summary: "Insurance lapsed",
				DetectedAt: timeutils.NowUnix(), Fingerprint: "fp" + label,
				Status: carrierintel.EventStatusOpen,
			}})
			if err != nil {
				return pulid.Nil, err
			}
			require.Len(t, rows, 1)

			return rows[0].ID, nil
		},
	}
	require.Len(t, gets, len(convertedGets)+1)

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	public := agentreadtest.NewReader(t, agentreadtest.Options{Engine: &publicRole{}, Resolvers: services})

	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")

	for name, c := range gets {
		t.Run(name, func(t *testing.T) {
			checkConvertedGet(t, name, c, &listReaders{
				reader: reader, public: public, engine: engine, mine: mine, theirs: theirs,
			}, nil)
		})
	}
}

// checkConvertedGet holds one converted get to row-level security: each of
// the tenant's records reads, withholding at the default agent data access
// only withheldAtInternal, another tenant's is not found, and at Public
// every withheld field is absent.
func checkConvertedGet(
	t *testing.T,
	name string,
	c *referenceCase,
	r *listReaders,
	withheldAtInternal []string,
) {
	t.Helper()

	reader, public, engine, mine, theirs := r.reader, r.public, r.engine, r.mine, r.theirs
	get, err := agentquerytoolservice.ConvertedGetTool(name, reader, engine)
	require.NoError(t, err)
	ours := writeReferenceRows(t, c, mine, "A")
	other := writeReferenceRows(t, c, theirs, "B")
	actor := mine.Actor
	call := func(id string) map[string]any {
		return agentquerytoolservice.ConvertedGetCall(get, pulid.ID(id))
	}

	t.Run("our record, as the card reads it", func(t *testing.T) {
		for _, id := range ours.rows {
			record := callTool(t, ours.ctx, get, actor, call(id))
			assert.Equal(t, id, record["id"])
			withheld, _ := record["withheldByAccess"].([]any)
			assertWithheld(t, withheldAtInternal, withheld)
		}
	})

	t.Run("only what the get documents is withheld at the default agent data access", func(t *testing.T) {
		data, err := get.Query(ours.ctx, &agentports.QueryToolParams{
			OrganizationID:    actor.OrganizationID,
			BusinessUnitID:    actor.BusinessUnitID,
			Actor:             actor,
			Timezone:          "America/Chicago",
			Params:            call(ours.rows[0]),
			DataAccessCeiling: permission.SensitivityInternal,
			RunID:             pulid.MustNew("ar_"),
		})
		require.NoError(t, err)
		withheld, _ := asMap(t, data)["withheldByAccess"].([]any)
		assertWithheld(t, withheldAtInternal, withheld)
	})

	t.Run("another tenant's record is not found", func(t *testing.T) {
		_, err := get.Query(ours.ctx, &agentports.QueryToolParams{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			Actor:          actor,
			Timezone:       "America/Chicago",
			Params:         call(other.rows[0]),
			RunID:          pulid.MustNew("ar_"),
		})
		require.Error(t, err)
	})

	t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
		hidden, err := agentquerytoolservice.ConvertedGetTool(name, public, &publicRole{})
		require.NoError(t, err)
		record := callTool(t, ours.ctx, hidden, actor, call(ours.rows[0]))
		withheld, _ := record["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		assert.Equal(t, ours.rows[0], record["id"])
		for _, field := range withheld {
			assert.NotContains(t, record, field)
		}
	})
}

// assertWithheld holds what a read withheld to what its get documents.
func assertWithheld(t *testing.T, want []string, withheld []any) {
	t.Helper()

	got := make([]string, 0, len(withheld))
	for _, field := range withheld {
		got = append(got, fmt.Sprint(field))
	}
	assert.ElementsMatch(t, want, got)
}
