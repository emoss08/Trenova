//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/tcasubscriptionrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablechangealertservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// alertTenant is one tenant's change alerts: the reader's own, and one a
// colleague in the same organization set up.
type alertTenant struct {
	ctx        context.Context
	alerts     []string
	colleagues string
}

func writeAlertRows(
	t *testing.T,
	admin *bun.DB,
	tenant *agentreadtest.Tenant,
	label string,
) *alertTenant {
	t.Helper()

	ctx := t.Context()
	org, bu := tenant.Actor.OrganizationID, tenant.Actor.BusinessUnitID

	tx, err := admin.BeginTx(ctx, nil)
	require.NoError(t, err)
	colleague := seedtest.NewUser(org, bu).
		WithUsername("watcher_"+strings.ToLower(label)+org.String()[4:12]).
		WithEmail("watcher_"+strings.ToLower(label)+org.String()[4:12]+"@example.com").
		Build(t, ctx, tx).ID
	require.NoError(t, tx.Commit())

	alert := func(owner pulid.ID, name, table string, status tablechangealert.SubscriptionStatus) string {
		entity := &tablechangealert.TCASubscription{
			ID:             pulid.MustNew("tcas_"),
			OrganizationID: org,
			BusinessUnitID: bu,
			UserID:         owner,
			Name:           name,
			TableName:      table,
			EventTypes:     []string{"UPDATE"},
			Conditions: []tablechangealert.Condition{
				{Field: "status", Operator: "eq", Value: "Delivered"},
				{Field: "total", Operator: "gt", Value: 5000},
			},
			ConditionMatch: "all",
			WatchedColumns: []string{"status"},
			CustomTitle:    "Heads up " + label,
			CustomMessage:  "Check the " + table + " change",
			Priority:       "medium",
			Status:         status,
		}
		_, insertErr := admin.NewInsert().Model(entity).Exec(ctx)
		require.NoError(t, insertErr)

		return entity.ID.String()
	}

	return &alertTenant{
		ctx: readscope.With(dbscope.WithTenant(ctx, tenant.Scope()), nil),
		alerts: []string{
			alert(tenant.Actor.UserID, "Big invoices "+label, "invoices", tablechangealert.SubscriptionStatusActive),
			alert(tenant.Actor.UserID, "Delivered "+label, "shipments", tablechangealert.SubscriptionStatusActive),
			alert(tenant.Actor.UserID, "Paused "+label, "shipments", tablechangealert.SubscriptionStatusPaused),
		},
		colleagues: alert(colleague, "Colleague "+label, "invoices", tablechangealert.SubscriptionStatusActive),
	}
}

// tcaRestrictedRole is a person whose role reaches Restricted fields.
type tcaRestrictedRole struct {
	mocks.DenyResourcesPermissionEngine
}

func (tcaRestrictedRole) GetResourcePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{
		MaxSensitivity: permission.SensitivityRestricted,
	}, nil
}

func (r tcaRestrictedRole) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	return permtest.Batch(ctx, r, userID, orgID, resources)
}

func alertQuery(
	ctx context.Context,
	tool agentports.AgentQueryTool,
	actor *permissionports.RequestActor,
	args map[string]any,
	ceiling permission.FieldSensitivity,
) (any, error) {
	return tool.Query(ctx, &agentports.QueryToolParams{
		OrganizationID: actor.OrganizationID, BusinessUnitID: actor.BusinessUnitID,
		Actor: actor, Timezone: "America/Chicago", Params: args,
		DataAccessCeiling: ceiling, RunID: pulid.MustNew("ar_"),
	})
}

// The change alert list reads, through the screen's resolver under row-level
// security, only the reader's own alerts: never a colleague's, never another
// tenant's. At Internal an alert shows its status, event types and condition
// match and withholds its name, table, watched columns and conditions; a
// role that reaches Restricted sees them, and the condition count, and may
// search; at Public everything but the id is withheld.
func TestConvertedLists_TableChangeAlertsUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	resolvers := agentread.ResolverParams{
		TableChangeAlertService: tablechangealertservice.New(tablechangealertservice.Params{
			Logger: logger,
			SubRepo: tcasubscriptionrepository.New(
				tcasubscriptionrepository.Params{DB: conn, Logger: logger}),
		}),
	}
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: resolvers})
	restrictedEngine := &tcaRestrictedRole{}
	restricted := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: restrictedEngine, Resolvers: resolvers,
	})
	public := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &publicRole{}, Resolvers: resolvers,
	})

	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	ours := writeAlertRows(t, admin, mine, "A")
	other := writeAlertRows(t, admin, theirs, "B")
	actor := mine.Actor
	const name = "report_list_table_change_alerts"

	alerts, err := agentquerytoolservice.ConvertedListTool(name, reader, engine)
	require.NoError(t, err)

	t.Run("only the reader's own alerts", func(t *testing.T) {
		ids, pages := walkByKey(t, ours.ctx, alerts, actor, nil, "id")
		assert.Greater(t, pages, 1, "the walk must cross a page")
		assert.ElementsMatch(t, ours.alerts, ids)
		assert.NotContains(t, ids, ours.colleagues, "a colleague's alerts are never listed")
		for _, id := range append(other.alerts, other.colleagues) {
			assert.NotContains(t, ids, id)
		}

		paused := rowsByID(callTool(t, ours.ctx, alerts, actor,
			map[string]any{"status": "Paused"}))
		assert.Len(t, paused, 1)
		assert.Contains(t, paused, ours.alerts[2])
	})

	t.Run("at Internal only status, event types and condition match show", func(t *testing.T) {
		data, queryErr := alertQuery(ours.ctx, alerts, actor, map[string]any{},
			permission.SensitivityInternal)
		require.NoError(t, queryErr)
		page := asMap(t, data)
		assert.ElementsMatch(t,
			[]any{"name", "tableName", "conditions", "watchedColumns"},
			page["withheldByAccess"])
		rows := rowsByID(page)
		require.Len(t, rows, len(ours.alerts))
		for _, row := range rows {
			assert.ElementsMatch(t,
				[]string{"id", "status", "eventTypes", "conditionMatch"}, keysOf(row),
				"nothing beyond the three Internal fields and the id")
		}

		_, queryErr = alertQuery(ours.ctx, alerts, actor, map[string]any{"query": "invoice"},
			permission.SensitivityInternal)
		require.ErrorContains(t, queryErr, "cannot search by query")
		_, queryErr = alertQuery(ours.ctx, alerts, actor, map[string]any{"tableName": "inv"},
			permission.SensitivityInternal)
		require.ErrorContains(t, queryErr, "hidden at your access")
	})

	t.Run("a role that reaches Restricted sees the rest and may search", func(t *testing.T) {
		reached, buildErr := agentquerytoolservice.ConvertedListTool(name, restricted,
			restrictedEngine)
		require.NoError(t, buildErr)
		data, queryErr := alertQuery(ours.ctx, reached, actor, map[string]any{},
			permission.SensitivityRestricted)
		require.NoError(t, queryErr)
		page := asMap(t, data)
		assert.Empty(t, page["withheldByAccess"])
		row := rowsByID(page)[ours.alerts[0]]
		assert.Equal(t, "Big invoices A", row["name"])
		assert.Equal(t, "invoices", row["tableName"])
		assert.InDelta(t, 2, row["conditionCount"], 0)
		assert.NotContains(t, row, "conditions")

		data, queryErr = alertQuery(ours.ctx, reached, actor, map[string]any{"query": "invoices"},
			permission.SensitivityRestricted)
		require.NoError(t, queryErr)
		assert.Equal(t, []string{ours.alerts[0]}, keysOf(rowsByID(asMap(t, data))))
	})

	t.Run("at Public everything but the id is withheld", func(t *testing.T) {
		hidden, buildErr := agentquerytoolservice.ConvertedListTool(name, public, engine)
		require.NoError(t, buildErr)
		page := callTool(t, ours.ctx, hidden, actor, map[string]any{})
		assert.NotEmpty(t, page["withheldByAccess"])
		for _, item := range page["items"].([]any) {
			assert.Equal(t, []string{"id"}, keysOf(item.(map[string]any)))
		}
	})
}
