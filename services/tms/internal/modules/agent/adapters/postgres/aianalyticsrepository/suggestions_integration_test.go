//go:build integration

package aianalyticsrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type suggestionFixture struct {
	ctx      context.Context
	db       *bun.DB
	tenant   pagination.TenantInfo
	dispatch pulid.ID
}

func (f suggestionFixture) agent(t *testing.T, name string) pulid.ID {
	t.Helper()

	definition := &agentdefinition.Definition{
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		Name:           name,
		Enabled:        true,
	}
	insert(f.ctx, t, f.db, definition)

	return definition.ID
}

func (f suggestionFixture) suggest(
	t *testing.T,
	agentID pulid.ID,
	outcome agentsuggestion.Outcome,
	createdAt int64,
) pulid.ID {
	t.Helper()

	current := f.dispatch
	row := &agentsuggestion.Suggestion{
		OrganizationID:   f.tenant.OrgID,
		BusinessUnitID:   f.tenant.BuID,
		UserID:           f.tenant.UserID,
		SuggestedAgentID: agentID,
		CurrentAgentID:   &current,
		Surface:          agentsuggestion.SurfaceHome,
		Outcome:          outcome,
	}
	insert(f.ctx, t, f.db, row)
	_, err := f.db.NewRaw(
		"UPDATE agent_suggestions SET created_at = ?, updated_at = ? WHERE id = ?",
		createdAt, createdAt, row.ID,
	).Exec(f.ctx)
	require.NoError(t, err)

	return row.ID
}

func (f suggestionFixture) stored(t *testing.T) []pulid.ID {
	t.Helper()

	var ids []pulid.ID
	require.NoError(t, f.db.NewSelect().
		Table("agent_suggestions").
		Column("id").
		Where("organization_id = ?", f.tenant.OrgID).
		Scan(f.ctx, &ids))

	return ids
}

func suggestionRowsByAgent(rows []aianalyticsports.SuggestionRow) map[pulid.ID]aianalyticsports.SuggestionRow {
	byAgent := make(map[pulid.ID]aianalyticsports.SuggestionRow, len(rows))
	for _, row := range rows {
		byAgent[row.AgentDefinitionID] = row
	}

	return byAgent
}

func TestSuggestionsCountEachOutcomeAndPassOverWhatWasLeftShown(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := suggestionFixture{ctx: ctx, db: db, tenant: seededTenant(ctx, t, db)}
	f.dispatch = f.agent(t, "Dispatch")
	receivables := f.agent(t, "Receivables")
	billing := f.agent(t, "Billing")
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	now := timeutils.NowUnix()
	pendingSince := aianalytics.SuggestionPendingSince(now)
	since := now - 24*3600

	f.suggest(t, receivables, agentsuggestion.OutcomeSwitched, now-600)
	f.suggest(t, receivables, agentsuggestion.OutcomeSentThrough, now-600)
	f.suggest(t, receivables, agentsuggestion.OutcomeAutoSwitched, now-600)
	f.suggest(t, receivables, agentsuggestion.OutcomeKept, now-600)
	f.suggest(t, receivables, agentsuggestion.OutcomePassedOver, now-600)
	f.suggest(t, receivables, agentsuggestion.OutcomeShown, pendingSince-60)
	f.suggest(t, receivables, agentsuggestion.OutcomeShown, pendingSince+60)
	f.suggest(t, receivables, agentsuggestion.OutcomeSwitched, since-3600)
	f.suggest(t, billing, agentsuggestion.OutcomeSwitched, now-60)
	f.suggest(t, billing, agentsuggestion.OutcomeSwitched, now-60)
	f.suggest(t, billing, agentsuggestion.OutcomeShown, now-60)

	rows, err := repo.Suggestions(ctx, &aianalyticsports.SuggestionRequest{
		Filter:       aianalyticsports.Filter{TenantInfo: f.tenant, Since: since},
		PendingSince: pendingSince,
	})
	require.NoError(t, err)
	byAgent := suggestionRowsByAgent(rows)
	require.Len(t, byAgent, 2)

	assert.Equal(t, aianalyticsports.SuggestionRow{
		AgentDefinitionID: receivables,
		AgentName:         "Receivables",
		SuggestionCounts: aianalyticsports.SuggestionCounts{
			Switched:     1,
			SentThrough:  1,
			AutoSwitched: 1,
			Kept:         1,
			PassedOver:   2,
		},
	}, byAgent[receivables], "a suggestion left shown past the window is passed over; one inside it is not counted yet")
	assert.Equal(t, aianalyticsports.SuggestionRow{
		AgentDefinitionID: billing,
		AgentName:         "Billing",
		SuggestionCounts:  aianalyticsports.SuggestionCounts{Switched: 2},
	}, byAgent[billing])

	onlyBilling, err := repo.Suggestions(ctx, &aianalyticsports.SuggestionRequest{
		Filter: aianalyticsports.Filter{
			TenantInfo:        f.tenant,
			Since:             since,
			AgentDefinitionID: billing,
		},
		PendingSince: pendingSince,
	})
	require.NoError(t, err)
	require.Len(t, onlyBilling, 1)
	assert.Equal(t, billing, onlyBilling[0].AgentDefinitionID)
}

func TestPruneDeletesSuggestionsOlderThanTheHorizon(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	f := suggestionFixture{ctx: ctx, db: db, tenant: seededTenant(ctx, t, db)}
	f.dispatch = f.agent(t, "Dispatch")
	receivables := f.agent(t, "Receivables")
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	now := timeutils.NowUnix()
	horizon := now - 90*24*3600
	f.suggest(t, receivables, agentsuggestion.OutcomeSwitched, horizon-1)
	f.suggest(t, receivables, agentsuggestion.OutcomeShown, horizon-3600)
	kept := f.suggest(t, receivables, agentsuggestion.OutcomeKept, horizon)
	recent := f.suggest(t, receivables, agentsuggestion.OutcomeSentThrough, now-60)

	require.NoError(t, repo.Prune(ctx, aianalyticsports.PruneRequest{
		TenantInfo: f.tenant,
		Before:     horizon,
	}))

	assert.ElementsMatch(t, []pulid.ID{kept, recent}, f.stored(t))
}
