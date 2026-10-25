package agentruneventrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func readEventsEveryWay(t *testing.T, reader agentports.RunReader) []string {
	t.Helper()

	byRun, _ := keeltest.Render(t, listQuery(&agentports.ListAgentRunEventsRequest{
		OwnerKind: string(agentports.RunStepOwnerAgentRun),
		OwnerID:   pulid.MustNew("ar_"),
		Reader:    reader,
	}).Limit(10))
	page, _ := keeltest.Render(t, agentdb.AgentRunEvents.Where(ofReadableRuns(reader)).Limit(10))

	return []string{byRun, page}
}

func TestRunEventReads_KeepAConversationRunsEventsToItsOwnerForAPerson(t *testing.T) {
	t.Parallel()

	for _, query := range readEventsEveryWay(t, agentports.PersonRunRead(pulid.MustNew("usr_"))) {
		assert.Contains(t, query, `are."owner_kind" <> $`, query)
		assert.Contains(t, query, `EXISTS (SELECT`, query)
		assert.Contains(t, query, `"agent_runs" event_run`, query)
		assert.Contains(t, query, `event_run."id" = are."owner_id"`, query)
		assert.Contains(t, query, `event_run."subject_type" <> $`, query)
		assert.Contains(t, query, `"assistant_threads" readable_thread`, query)
		assert.Contains(t, query, `readable_thread."id" = event_run."subject_id"`, query)
		assert.Contains(t, query, `readable_thread."user_id" = $`, query)
	}
}

func TestRunEventReads_ShowNoConversationRunsEventsToAReaderWhoIsNoPerson(t *testing.T) {
	t.Parallel()

	for _, query := range readEventsEveryWay(t, agentports.PersonRunRead(pulid.Nil)) {
		assert.Contains(t, query, `event_run."subject_type" <> $`, query)
		assert.NotContains(t, query, "assistant_threads", query)
	}
}

func TestRunEventReads_LeaveAnUnattendedReadAsItWas(t *testing.T) {
	t.Parallel()

	for _, query := range readEventsEveryWay(t, agentports.UnattendedRunRead()) {
		assert.NotContains(t, query, "agent_runs", query)
		assert.NotContains(t, query, "assistant_threads", query)
	}
}
