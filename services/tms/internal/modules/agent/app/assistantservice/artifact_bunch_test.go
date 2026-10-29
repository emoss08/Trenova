package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func bunchingRecorder(
	t *testing.T,
) (*artifactrecorder.Recorder, *assistanttest.BunchingRepo, pagination.TenantInfo) {
	t.Helper()

	repo := &assistanttest.BunchingRepo{}
	svc := &Service{logger: zap.NewNop(), artifacts: repo}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return svc.newArtifactRecorder(
		t.Context(),
		thread,
		tenant,
		assistanttest.Actor(),
		nil,
	), repo, tenant
}

func TestBunch_ASingleGetStillMakesItsCard(t *testing.T) {
	t.Parallel()

	recorder, repo, _ := bunchingRecorder(t)

	shown, err := recorder.Observe(assistanttest.InvoiceRead("call_1", "INV-1"))
	require.NoError(t, err)
	require.NotNil(t, shown)
	assert.Equal(t, string(assistantartifact.KindEntityCard), shown.Kind)
	assert.Empty(t, repo.ListedBy, "a first read has nothing to bunch with")
}

func TestBunch_SeveralGetsOfOneToolInATurnBecomeOneTable(t *testing.T) {
	t.Parallel()

	first, repo, tenant := bunchingRecorder(t)
	_, err := first.Observe(assistanttest.InvoiceRead("call_1", "INV-1"))
	require.NoError(t, err)
	card := repo.Current()[0]

	second := first
	shown, err := second.Observe(assistanttest.InvoiceRead("call_2", "INV-2", "call_1"))
	require.NoError(t, err)
	require.NotNil(t, shown)
	assert.Equal(t, string(assistantartifact.KindTableView), shown.Kind)
	assert.Equal(t, []pulid.ID{card.ID}, repo.Deleted, "the first card is folded into the table")
	assert.Equal(t, []pulid.ID{card.ID}, shown.Replaces, "the model is told which card is gone")
	require.Len(t, repo.ListedBy, 1)
	assert.Equal(t, tenant, repo.ListedBy[0].TenantInfo)
	assert.Equal(t, []string{"call_1"}, repo.ListedBy[0].CallIDs)

	_, err = second.Observe(assistanttest.InvoiceRead("call_3", "INV-3", "call_1", "call_2"))
	require.NoError(t, err)

	live := repo.Current()
	require.Len(t, live, 1, "one table for the turn and tool, not a card per call")
	table := live[0]
	assert.Equal(t, assistantartifact.KindTableView, table.Kind)
	assert.Equal(t, "call_1", table.SourceToolCallID, "keyed by the turn's first call of the tool")
	assert.Equal(t, "Invoices (3)", table.Title)
	assert.Equal(t, true, table.Payload[toolartifacts.PayloadBunched])
	assert.Equal(
		t,
		[]string{"call_1", "call_2", "call_3"},
		table.Payload[toolartifacts.PayloadCalls],
	)
	assert.Equal(t, "invoice", table.Payload["recordEntity"])

	rows := table.Payload[artifactpayload.PayloadRows].([]any)
	require.Len(t, rows, 3)
	numbers := make([]any, 0, 3)
	for _, row := range rows {
		record := row.(map[string]any)
		numbers = append(numbers, record["number"])
		assert.NotEmpty(t, record[artifactpayload.RecordIDKey], "each row opens its invoice")
	}
	assert.Equal(t, []any{"INV-1", "INV-2", "INV-3"}, numbers)
}

func TestBunch_AnEarlierCallWithNoCardKeepsTheNewCard(t *testing.T) {
	t.Parallel()

	recorder, repo, _ := bunchingRecorder(t)
	_, err := recorder.Observe(assistanttest.InvoiceRead("call_1", "INV-1"))
	require.NoError(t, err)

	other := assistanttest.InvoiceRead("call_2", "INV-2", "call_9")
	shown, err := recorder.Observe(other)
	require.NoError(t, err)
	assert.Equal(t, string(assistantartifact.KindEntityCard), shown.Kind)
	assert.Empty(t, repo.Deleted)
}

func TestBunch_TheTurnKeepsTheTableNotTheCardsItReplaced(t *testing.T) {
	t.Parallel()

	recorder, repo, _ := bunchingRecorder(t)
	_, err := recorder.Observe(assistanttest.InvoiceRead("call_1", "INV-1"))
	require.NoError(t, err)
	card := repo.Current()[0]
	_, err = recorder.Observe(assistanttest.InvoiceRead("call_2", "INV-2", "call_1"))
	require.NoError(t, err)
	table := repo.Current()[0]

	finish, _, _ := bunchingRecorder(t)
	finish.Adopt([]*assistantartifact.Artifact{card, table})

	require.Len(t, finish.Recorded, 1)
	assert.Equal(t, table.ID, finish.Recorded[0].ID)
}

func TestBunch_TheCardsATableReplacedAreWithdrawnFromTheReader(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.BunchingRepo{}
	svc := &Service{logger: zap.NewNop(), artifacts: repo}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	var events []kernelports.StreamEvent
	recorder := svc.newArtifactRecorder(t.Context(), thread, tenant, assistanttest.Actor(),
		func(event kernelports.StreamEvent) { events = append(events, event) })

	_, err := recorder.Observe(assistanttest.InvoiceRead("call_1", "INV-1"))
	require.NoError(t, err)
	card := repo.Current()[0]
	_, err = recorder.Observe(assistanttest.InvoiceRead("call_2", "INV-2", "call_1"))
	require.NoError(t, err)
	table := repo.Current()[0]

	removed := make([]pulid.ID, 0, 1)
	for _, event := range events {
		if event.Event == assistantstreamports.AssistantEventArtifactRemoved {
			data, ok := event.Data.(assistantstreamports.AssistantArtifactRemovedEvent)
			require.True(t, ok)
			removed = append(removed, data.ID)
		}
	}
	assert.Equal(t, []pulid.ID{card.ID}, removed,
		"a reader watching the turn is told the card left, or it keeps it beside the table")

	require.Len(t, recorder.Recorded, 1, "the turn reports the table, not the card it replaced")
	assert.Equal(t, table.ID, recorder.Recorded[0].ID)
}
