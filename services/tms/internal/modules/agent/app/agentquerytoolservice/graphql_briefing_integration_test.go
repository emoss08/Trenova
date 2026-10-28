//go:build integration

package agentquerytoolservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/briefingrepository"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/briefingservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const briefingDay = "2026-10-23"

func seededBriefing(tenant *agentreadtest.Tenant) *briefing.Briefing {
	return &briefing.Briefing{
		OrganizationID: tenant.Actor.OrganizationID,
		BusinessUnitID: tenant.Actor.BusinessUnitID,
		RoleKey:        briefing.RoleGeneral,
		BriefingDate:   briefingDay,
		Status:         briefing.StatusReady,
		Headline:       "Four service failures and $4,210 received yesterday.",
		Narrated:       true,
		Sections: []briefing.Section{
			{
				Key: briefing.SectionAttention, Title: "Needs attention",
				Summary: "12 items open.", Body: "Twelve things want you.",
				Items: []briefing.Item{
					{
						Label: "Service failure",
						Value: "4",
						Path:  "/desk/watchtower?kinds=ServiceFailure",
					},
					{
						Label: "EDI quarantined",
						Value: "8",
						Path:  "/desk/watchtower?kinds=EDIInboundQuarantined",
					},
				},
			},
			{
				Key: briefing.SectionCash, Title: "Cash received yesterday",
				Summary: "2 payments posted yesterday.", Body: "Two payments came in.",
				Items: []briefing.Item{{Label: "Received in USD", Value: "4210.00"}},
			},
		},
		Facts: map[string]any{},
	}
}

// The morning's page is read through todaysBriefing under row-level
// security, cut by the briefing service to the sections the reader may read:
// the whole narrated page for a reader of everything, and for a reader
// missing a section's permission the rest of it in its computed wording,
// with what was left out named.
func TestConvertedGets_DailyBriefingIsCutToTheReader(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	_, resolvers := convertedListCases(conn, admin)
	logger := zap.NewNop()
	repo := briefingrepository.New(briefingrepository.Params{DB: conn, Logger: logger})

	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	mineCtx := readscope.With(dbscope.WithTenant(t.Context(), mine.Scope()), nil)
	theirCtx := readscope.With(dbscope.WithTenant(t.Context(), theirs.Scope()), nil)
	_, err := repo.Upsert(mineCtx, seededBriefing(mine))
	require.NoError(t, err)

	tool := func(t *testing.T, engine permissionports.PermissionEngine) agenttoolports.AgentQueryTool {
		t.Helper()
		resolvers.BriefingService = briefingservice.New(briefingservice.Params{
			Logger: logger, Repo: repo, Permissions: engine,
		})
		reader := agentreadtest.NewReader(
			t,
			agentreadtest.Options{Engine: engine, Resolvers: resolvers},
		)
		get, getErr := agentquerytoolservice.ConvertedGetTool(
			"insight_get_daily_briefing",
			reader,
			engine,
		)
		require.NoError(t, getErr)

		return get
	}

	t.Run("a reader of every section reads the narrated page", func(t *testing.T) {
		page := callTool(t, mineCtx, tool(t, &mocks.DenyResourcesPermissionEngine{}), mine.Actor,
			map[string]any{"date": briefingDay})

		assert.Equal(t, "Four service failures and $4,210 received yesterday.", page["headline"])
		assert.Equal(t, true, page["narrated"])
		assert.Empty(t, page["withheldSections"])
		assert.NotContains(t, page, "note")
		sections, _ := page["sections"].([]any)
		require.Len(t, sections, 2)
		attention, _ := sections[0].(map[string]any)
		assert.Equal(t, "Twelve things want you.", attention["read"])
	})

	t.Run("a reader missing cash reads the rest in its computed wording", func(t *testing.T) {
		engine := &mocks.DenyResourcesPermissionEngine{
			Denied: []string{permission.ResourceCustomerPayment.String()},
		}
		page := callTool(
			t,
			mineCtx,
			tool(t, engine),
			mine.Actor,
			map[string]any{"date": briefingDay},
		)

		assert.Empty(t, page["headline"], "the headline cites the cash")
		assert.Equal(t, false, page["narrated"])
		assert.Equal(t, []any{map[string]any{
			"key": "Cash", "title": "Cash received yesterday", "partial": false,
		}}, page["withheldSections"])
		assert.NotEmpty(t, page["note"])
		sections, _ := page["sections"].([]any)
		require.Len(t, sections, 1)
		attention, _ := sections[0].(map[string]any)
		assert.Equal(t, "Attention", attention["key"])
		assert.Equal(t, "12 items open.", attention["read"])
	})

	t.Run("a reader missing one watchtower kind keeps the lines they may read", func(t *testing.T) {
		engine := &mocks.DenyResourcesPermissionEngine{
			Denied: []string{permission.ResourceEDI.String()},
		}
		page := callTool(
			t,
			mineCtx,
			tool(t, engine),
			mine.Actor,
			map[string]any{"date": briefingDay},
		)

		sections, _ := page["sections"].([]any)
		require.Len(t, sections, 2)
		attention, _ := sections[0].(map[string]any)
		items, _ := attention["items"].([]any)
		require.Len(t, items, 1)
		item, _ := items[0].(map[string]any)
		assert.Equal(t, "Service failure", item["label"])
		assert.Empty(t, attention["summary"], "its wording counted the EDI files too")
		assert.Equal(t, []any{map[string]any{
			"key": "Attention", "title": "Needs attention", "partial": true,
		}}, page["withheldSections"])
	})

	t.Run("another tenant has no page", func(t *testing.T) {
		page := callTool(t, theirCtx, tool(t, &mocks.DenyResourcesPermissionEngine{}), theirs.Actor,
			map[string]any{"date": briefingDay})

		assert.Equal(t, false, page["found"])
		assert.NotContains(t, page, "headline")
	})

	t.Run("nothing the page selects is withheld at Internal", func(t *testing.T) {
		get := tool(t, &mocks.DenyResourcesPermissionEngine{})
		data, queryErr := get.Query(mineCtx, &agenttoolports.QueryToolParams{
			OrganizationID: mine.Actor.OrganizationID, BusinessUnitID: mine.Actor.BusinessUnitID,
			Actor: mine.Actor, Timezone: "America/Chicago",
			DataAccessCeiling: permission.SensitivityInternal,
			Params:            map[string]any{"date": briefingDay}, RunID: pulid.MustNew("ar_"),
		})
		require.NoError(t, queryErr)
		assert.NotContains(t, asMap(t, data), "withheldByAccess")
	})
}
