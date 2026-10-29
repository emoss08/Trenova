//go:build integration

package agentruneventrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunevent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func eventOwners(events []*agentrunevent.AgentRunEvent) []pulid.ID {
	owners := make([]pulid.ID, 0, len(events))
	for _, event := range events {
		owners = append(owners, event.OwnerID)
	}

	return owners
}

func TestRunEventReads_KeepAConversationRunsEventsToItsOwner(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
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
	require.NoError(t, conversationdb.Threads.Create(ctx, db, thread))

	newRun := func(subject agent.SubjectType, subjectID pulid.ID) *agent.AgentRun {
		t.Helper()
		run := &agent.AgentRun{
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
			StartedAt:        now,
		}
		require.NoError(t, agentdb.AgentRuns.Create(ctx, db, run))

		return run
	}
	conversationRun := newRun(agent.SubjectAssistantThread, thread.ID)
	shipmentRun := newRun(agent.SubjectShipment, pulid.MustNew("shp_"))

	runOwner := string(agentruntimeports.RunStepOwnerAgentRun)
	events := make([]*agentrunevent.AgentRunEvent, 0, 2)
	for _, run := range []*agent.AgentRun{conversationRun, shipmentRun} {
		events = append(events, &agentrunevent.AgentRunEvent{
			ID:             pulid.MustNew("are_"),
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			OwnerKind:      runOwner,
			OwnerID:        run.ID,
			Sequence:       1,
			Kind:           "message",
			Payload:        map[string]any{"content": "what the run said"},
			OccurredAt:     now,
		})
	}
	require.NoError(t, repo.Append(ctx, agentruntimeports.AppendAgentRunEventsRequest{
		TenantInfo: tenant,
		Events:     events,
	}))

	readers := map[string]struct {
		reader agentruntimeports.RunReader
		sees   []pulid.ID
	}{
		"the owner": {
			reader: agentruntimeports.PersonRunRead(owner),
			sees:   []pulid.ID{conversationRun.ID, shipmentRun.ID},
		},
		"a colleague": {
			reader: agentruntimeports.PersonRunRead(pulid.MustNew("usr_")),
			sees:   []pulid.ID{shipmentRun.ID},
		},
		"an agent or an API key": {
			reader: agentruntimeports.PersonRunRead(pulid.Nil),
			sees:   []pulid.ID{shipmentRun.ID},
		},
		"the runtime": {
			reader: agentruntimeports.UnattendedRunRead(),
			sees:   []pulid.ID{conversationRun.ID, shipmentRun.ID},
		},
	}

	for name, tc := range readers {
		page, listErr := repo.ListConnection(
			ctx,
			&agentruntimeports.ListAgentRunEventConnectionRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: tenant,
					Pagination: pagination.Info{Limit: 10},
				},
				Cursor:    pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
				OwnerKind: runOwner,
				Reader:    tc.reader,
			},
		)
		require.NoError(t, listErr, name)
		assert.ElementsMatch(t, tc.sees, eventOwners(page.Items), name)
		require.NotNil(t, page.TotalCount, name)
		assert.Equal(t, len(tc.sees), *page.TotalCount,
			"%s: the count leaves out what the page leaves out", name)

		for _, run := range []*agent.AgentRun{conversationRun, shipmentRun} {
			byRun, runErr := repo.List(ctx, agentruntimeports.ListAgentRunEventsRequest{
				TenantInfo: tenant,
				OwnerKind:  runOwner,
				OwnerID:    run.ID,
				Reader:     tc.reader,
			})
			require.NoError(t, runErr, name)
			want := []pulid.ID{}
			for _, seen := range tc.sees {
				if seen == run.ID {
					want = append(want, run.ID)
				}
			}
			assert.ElementsMatch(t, want, eventOwners(byRun),
				"%s: a run they may not see has no events to read", name)
		}
	}
}
