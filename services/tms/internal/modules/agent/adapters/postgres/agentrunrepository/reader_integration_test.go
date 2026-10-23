//go:build integration

package agentrunrepository

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func storedRunIDs(runs []*agent.AgentRun) []pulid.ID {
	ids := make([]pulid.ID, 0, len(runs))
	for _, run := range runs {
		ids = append(ids, run.ID)
	}

	return ids
}

func TestRunReads_KeepAConversationsRunsToItsOwner(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	owner := data.User.ID
	now := timeutils.NowUnix()

	thread := &conversation.Thread{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		UserID:            owner,
		AgentDefinitionID: pulid.MustNew("agd_"),
		Title:             "Private conversation",
	}
	_, err := db.NewInsert().Model(thread).Exec(ctx)
	require.NoError(t, err)

	newRun := func(subject agent.SubjectType, subjectID pulid.ID, startedAt int64) *agent.AgentRun {
		t.Helper()
		run, createErr := repo.Create(ctx, &agent.AgentRun{
			ID:               pulid.MustNew("ar_"),
			OrganizationID:   tenant.OrgID,
			BusinessUnitID:   tenant.BuID,
			AgentType:        agent.TypeGeneral,
			Trigger:          agent.RunTriggerChat,
			SubjectType:      subject,
			SubjectID:        subjectID,
			Status:           agent.RunStatusCompleted,
			PromptVersion:    "v1",
			InputContextHash: "hash",
			StartedAt:        startedAt,
		})
		require.NoError(t, createErr)

		return run
	}
	conversationRun := newRun(agent.SubjectAssistantThread, thread.ID, now-30)
	orphanRun := newRun(agent.SubjectAssistantThread, pulid.MustNew("athr_"), now-20)
	shipmentRun := newRun(agent.SubjectShipment, pulid.MustNew("shp_"), now-10)
	all := []pulid.ID{conversationRun.ID, orphanRun.ID, shipmentRun.ID}

	readers := map[string]struct {
		reader agentports.RunReader
		sees   []pulid.ID
	}{
		"the owner": {
			reader: agentports.PersonRunRead(owner),
			sees:   []pulid.ID{conversationRun.ID, shipmentRun.ID},
		},
		"a colleague": {
			reader: agentports.PersonRunRead(pulid.MustNew("usr_")),
			sees:   []pulid.ID{shipmentRun.ID},
		},
		"an agent or an API key": {
			reader: agentports.PersonRunRead(pulid.Nil),
			sees:   []pulid.ID{shipmentRun.ID},
		},
		"the runtime": {
			reader: agentports.UnattendedRunRead(),
			sees:   all,
		},
	}

	for name, tc := range readers {
		listed, listErr := repo.List(ctx, &agentports.ListAgentRunRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenant,
				Pagination: pagination.Info{Limit: 10},
			},
			Reader: tc.reader,
		})
		require.NoError(t, listErr, name)
		assert.ElementsMatch(t, tc.sees, storedRunIDs(listed.Items), name)
		assert.Equal(t, len(tc.sees), listed.Total, name)

		connection, connErr := repo.ListConnection(ctx, &agentports.ListAgentRunConnectionRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenant,
				Pagination: pagination.Info{Limit: 10},
			},
			Cursor: pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
			Reader: tc.reader,
		})
		require.NoError(t, connErr, name)
		assert.ElementsMatch(t, tc.sees, storedRunIDs(connection.Items), name)
		require.NotNil(t, connection.TotalCount, name)
		assert.Equal(t, len(tc.sees), *connection.TotalCount,
			"%s: the count leaves out what the page leaves out", name)

		byIDs, idsErr := repo.ListByIDs(ctx, agentports.ListAgentRunsByIDsRequest{
			IDs:        all,
			TenantInfo: tenant,
			Reader:     tc.reader,
		})
		require.NoError(t, idsErr, name)
		assert.ElementsMatch(t, tc.sees, storedRunIDs(byIDs), name)

		transcripts, trErr := repo.ListTranscriptsByIDs(ctx, agentports.ListAgentRunsByIDsRequest{
			IDs:        all,
			TenantInfo: tenant,
			Reader:     tc.reader,
		})
		require.NoError(t, trErr, name)
		assert.ElementsMatch(t, tc.sees, storedRunIDs(transcripts), name)

		for _, id := range all {
			got, getErr := repo.GetByID(ctx, agentports.GetAgentRunByIDRequest{
				ID:         id,
				TenantInfo: &tenant,
				Reader:     tc.reader,
			})
			if slices.Contains(tc.sees, id) {
				require.NoError(t, getErr, name)
				assert.Equal(t, id, got.ID, name)

				continue
			}
			require.Error(t, getErr, name)
			assert.True(t, errortypes.IsNotFoundError(getErr),
				"%s: a run they may not see reads as missing", name)
		}
	}

	_, err = repo.GetByID(ctx, agentports.GetAgentRunByIDRequest{
		ID:         pulid.MustNew("ar_"),
		TenantInfo: &tenant,
		Reader:     agentports.PersonRunRead(owner),
	})
	assert.True(t, errortypes.IsNotFoundError(err), "a missing id reads the same way")
}
