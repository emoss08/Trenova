package agentrunrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func readEveryWay(t *testing.T, reader agentruntimeports.RunReader) []string {
	t.Helper()

	run := agentdb.AgentRuns
	runID := pulid.MustNew("arun_")
	byID, _ := keeltest.Render(t, readableRuns(reader).Where(run.ID.Eq(runID)).Limit(1))
	byIDs, _ := keeltest.Render(t, readableRuns(reader).Where(run.ID.In(runID)).Limit(1))
	transcript, _ := keeltest.Render(t, transcripts([]pulid.ID{runID}, reader))

	return []string{byID, byIDs, transcript}
}

func TestRunReads_LeaveTheTranscriptBehind(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, readableRuns(agentruntimeports.UnattendedRunRead()).Limit(1))

	assert.NotContains(t, sql, "transcript")
	assert.Contains(t, sql, `ar."summary"`, "the run itself is read")
}

func TestListTranscriptsByIDs_ReadsOnlyTheTranscriptOfTheTenantsRuns(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, transcripts(
		[]pulid.ID{pulid.MustNew("arun_")},
		agentruntimeports.UnattendedRunRead(),
	))

	assert.Contains(t, sql, `ar."transcript"`)
	assert.NotContains(t, sql, `ar."summary"`, "nothing but the transcript and its keys")
	assert.Contains(t, sql, `ar."organization_id"`)
	assert.Contains(t, sql, `ar."business_unit_id"`)
	assert.Contains(t, sql, `ar."organization_id" = $`)
}

func TestRunReads_KeepConversationRunsToTheirOwnerForAPerson(t *testing.T) {
	t.Parallel()

	for _, query := range readEveryWay(t, agentruntimeports.PersonRunRead(pulid.MustNew("usr_"))) {
		assert.Contains(t, query, `ar."subject_type" <> 'AssistantThread'`, query)
		assert.Contains(t, query, `EXISTS (SELECT`, query)
		assert.Contains(t, query, `"assistant_threads" readable_thread`, query)
		assert.Contains(t, query, `readable_thread."id" = ar."subject_id"`, query)
		assert.Contains(t, query, `readable_thread."user_id" = $`, query)
	}
}

func TestRunReads_ShowNoConversationRunsToAReaderWhoIsNoPerson(t *testing.T) {
	t.Parallel()

	for _, query := range readEveryWay(t, agentruntimeports.PersonRunRead(pulid.Nil)) {
		assert.Contains(t, query, `ar."subject_type" <> 'AssistantThread'`, query)
		assert.NotContains(t, query, "assistant_threads", query)
	}
}

func TestRunReads_LeaveAnUnattendedReadAsItWas(t *testing.T) {
	t.Parallel()

	for _, query := range readEveryWay(t, agentruntimeports.UnattendedRunRead()) {
		assert.NotContains(t, query, `ar."subject_type" <> 'AssistantThread'`, query)
		assert.NotContains(t, query, "assistant_threads", query)
	}
}
