//go:build integration

package agentsuggestionrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion/agentsuggestiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentsuggestionports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tenantFixture struct {
	info      pagination.TenantInfo
	suggested pulid.ID
	current   pulid.ID
}

type harness struct {
	ctx       context.Context
	admin     *keel.DB
	conn      *postgres.Connection
	repo      agentsuggestionports.AgentSuggestionRepository
	home      tenantFixture
	colleague pulid.ID
	other     tenantFixture
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	h := &harness{
		ctx:   ctx,
		admin: admin,
		home: tenantFixture{info: pagination.TenantInfo{
			OrgID:  data.Organization.ID,
			BuID:   data.BusinessUnit.ID,
			UserID: data.User.ID,
		}},
	}

	h.colleague = seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("suggestion_colleague").
		WithEmail("suggestion.colleague@example.com").
		Build(t, ctx, admin).ID

	bu := seedtest.NewBusinessUnit().WithCode("OTHERBU").Build(t, ctx, admin)
	org := seedtest.NewOrganization(bu.ID, data.State.ID).
		WithName("Other Organization").
		WithScacCode("OTHR").
		WithDOTNumber("7654321").
		WithBucketName("other-organization").
		Build(t, ctx, admin)
	user := seedtest.NewUser(org.ID, bu.ID).
		WithUsername("suggestion_other").
		WithEmail("suggestion.other@example.com").
		Build(t, ctx, admin)
	h.other.info = pagination.TenantInfo{OrgID: org.ID, BuID: bu.ID, UserID: user.ID}

	h.home.suggested = h.agent(t, h.home.info, "Receivables")
	h.home.current = h.agent(t, h.home.info, "Dispatch")
	h.other.suggested = h.agent(t, h.other.info, "Billing")
	h.other.current = h.agent(t, h.other.info, "Dispatch")

	h.conn = dbtest.NewRLSConnection(t, admin, dsn)
	h.repo = New(Params{DB: h.conn, Logger: zap.NewNop()})

	return h
}

func (h *harness) agent(t *testing.T, tenant pagination.TenantInfo, name string) pulid.ID {
	t.Helper()

	definition := &agentdefinition.Definition{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Name:           name,
		Enabled:        true,
	}
	require.NoError(t, agentdefinitiondb.Definitions.Create(h.scoped(tenant), h.admin, definition))

	return definition.ID
}

func (h *harness) scoped(tenant pagination.TenantInfo) context.Context {
	return dbscope.WithTenant(h.ctx, tenant.DBTenant())
}

func (h *harness) shown(tenant tenantFixture) *agentsuggestion.Suggestion {
	current := tenant.current

	return &agentsuggestion.Suggestion{
		OrganizationID:   tenant.info.OrgID,
		BusinessUnitID:   tenant.info.BuID,
		UserID:           tenant.info.UserID,
		SuggestedAgentID: tenant.suggested,
		CurrentAgentID:   &current,
		Surface:          agentsuggestion.SurfaceHome,
		Outcome:          agentsuggestion.OutcomeShown,
	}
}

func (h *harness) insert(t *testing.T, tenant tenantFixture) *agentsuggestion.Suggestion {
	t.Helper()

	inserted, err := h.repo.Insert(h.scoped(tenant.info), h.shown(tenant))
	require.NoError(t, err)

	return inserted
}

func (h *harness) stored(
	t *testing.T,
	tenant tenantFixture,
	id pulid.ID,
) *agentsuggestion.Suggestion {
	t.Helper()

	row, err := agentsuggestiondb.Suggestions.Get(h.scoped(tenant.info), h.admin, id)
	require.NoError(t, err)

	return row
}

func (h *harness) visible(t *testing.T, tenant pagination.TenantInfo) []pulid.ID {
	t.Helper()

	ids, err := dbtx.Read(h.scoped(tenant), h.conn, func(ctx context.Context) ([]pulid.ID, error) {
		return keel.QueryAll(
			ctx,
			h.conn.Keel(ctx),
			pgx.RowTo[pulid.ID],
			"SELECT id FROM agent_suggestions",
		)
	})
	require.NoError(t, err)

	return ids
}

func TestInsertThenSettleRecordsWhatThePersonDid(t *testing.T) {
	h := newHarness(t)

	inserted := h.insert(t, h.home)
	require.False(t, inserted.ID.IsNil())
	assert.Equal(t, agentsuggestion.OutcomeShown, h.stored(t, h.home, inserted.ID).Outcome)

	settled, err := h.repo.Settle(h.scoped(h.home.info), &agentsuggestionports.SettleAgentSuggestionRow{
		TenantInfo: h.home.info,
		ID:         inserted.ID,
		UserID:     h.home.info.UserID,
		Outcome:    agentsuggestion.OutcomeSwitched,
	})
	require.NoError(t, err)
	assert.Equal(t, inserted.ID, settled.ID)
	assert.Equal(t, agentsuggestion.OutcomeSwitched, settled.Outcome)
	assert.Equal(t, int64(1), settled.Version)
	assert.Equal(t, h.home.suggested, settled.SuggestedAgentID)
	require.NotNil(t, settled.CurrentAgentID)
	assert.Equal(t, h.home.current, *settled.CurrentAgentID)

	stored := h.stored(t, h.home, inserted.ID)
	assert.Equal(t, agentsuggestion.OutcomeSwitched, stored.Outcome)
	assert.Equal(t, int64(1), stored.Version)
}

func TestSettleLeavesAnAlreadySettledSuggestionAsItWas(t *testing.T) {
	h := newHarness(t)
	inserted := h.insert(t, h.home)
	ctx := h.scoped(h.home.info)

	first, err := h.repo.Settle(ctx, &agentsuggestionports.SettleAgentSuggestionRow{
		TenantInfo: h.home.info,
		ID:         inserted.ID,
		UserID:     h.home.info.UserID,
		Outcome:    agentsuggestion.OutcomeKept,
	})
	require.NoError(t, err)

	again, err := h.repo.Settle(ctx, &agentsuggestionports.SettleAgentSuggestionRow{
		TenantInfo: h.home.info,
		ID:         inserted.ID,
		UserID:     h.home.info.UserID,
		Outcome:    agentsuggestion.OutcomeSwitched,
	})
	require.NoError(t, err)
	assert.Equal(t, agentsuggestion.OutcomeKept, again.Outcome)
	assert.Equal(t, first.Version, again.Version)
	assert.Equal(t, first.UpdatedAt, again.UpdatedAt)

	stored := h.stored(t, h.home, inserted.ID)
	assert.Equal(t, agentsuggestion.OutcomeKept, stored.Outcome)
	assert.Equal(t, int64(1), stored.Version)
}

func TestSettleRefusesASuggestionShownToSomeoneElse(t *testing.T) {
	h := newHarness(t)
	inserted := h.insert(t, h.home)

	colleague := h.home.info
	colleague.UserID = h.colleague
	_, err := h.repo.Settle(h.scoped(colleague), &agentsuggestionports.SettleAgentSuggestionRow{
		TenantInfo: colleague,
		ID:         inserted.ID,
		UserID:     h.colleague,
		Outcome:    agentsuggestion.OutcomeSwitched,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "got %v", err)

	stored := h.stored(t, h.home, inserted.ID)
	assert.Equal(t, agentsuggestion.OutcomeShown, stored.Outcome)
	assert.Equal(t, int64(0), stored.Version)
}

func TestRowLevelSecurityKeepsEachTenantsSuggestionsApart(t *testing.T) {
	h := newHarness(t)
	home := h.insert(t, h.home)
	other := h.insert(t, h.other)

	assert.ElementsMatch(t, []pulid.ID{home.ID}, h.visible(t, h.home.info))
	assert.ElementsMatch(t, []pulid.ID{other.ID}, h.visible(t, h.other.info))

	_, err := h.repo.Settle(h.scoped(h.home.info), &agentsuggestionports.SettleAgentSuggestionRow{
		TenantInfo: h.other.info,
		ID:         other.ID,
		UserID:     h.other.info.UserID,
		Outcome:    agentsuggestion.OutcomeSwitched,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "got %v", err)
	assert.Equal(t, agentsuggestion.OutcomeShown, h.stored(t, h.other, other.ID).Outcome)

	_, err = h.repo.Insert(h.scoped(h.home.info), h.shown(h.other))
	require.Error(t, err, "a row for another tenant is refused by the policy")
	assert.ElementsMatch(t, []pulid.ID{other.ID}, h.visible(t, h.other.info))

	settled, err := h.repo.Settle(h.scoped(h.other.info), &agentsuggestionports.SettleAgentSuggestionRow{
		TenantInfo: h.other.info,
		ID:         other.ID,
		UserID:     h.other.info.UserID,
		Outcome:    agentsuggestion.OutcomeSwitched,
	})
	require.NoError(t, err)
	assert.Equal(t, agentsuggestion.OutcomeSwitched, settled.Outcome)
}
