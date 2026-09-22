package assistantservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubArtifactRepo struct {
	repositories.AssistantArtifactRepository

	upserts []*assistantartifact.Artifact
	listed  []*assistantartifact.Artifact
}

func (r *stubArtifactRepo) Upsert(
	_ context.Context,
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	if artifact.ID.IsNil() {
		artifact.ID = pulid.MustNew("art_")
	}
	copied := *artifact
	r.upserts = append(r.upserts, &copied)

	return artifact, nil
}

func (r *stubArtifactRepo) ListByThread(
	_ context.Context,
	_ repositories.ListArtifactsRequest,
) ([]*assistantartifact.Artifact, error) {
	return r.listed, nil
}

func observation(name string, data any) serviceports.ToolObservation {
	return serviceports.ToolObservation{
		Call: serviceports.ToolCall{ID: "call_" + name, Name: name},
		Data: data,
	}
}

// A preview is read by a person as a table, not as a fenced JSON blob the
// model was handed. The artifact carries the rows and columns as the tool
// published them, named after the report.
func TestArtifactFromObservation_PreviewBecomesATable(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("preview_report", map[string]any{
		"name":     "Revenue by customer",
		"dataset":  "shipments",
		"columns":  []any{map[string]any{"id": "customer", "label": "Customer", "type": "string"}},
		"rowCount": 2,
		"rows":     []any{map[string]any{"Customer": "Acme"}, map[string]any{"Customer": "Globex"}},
	}))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindReportPreview, artifact.Kind)
	assert.Equal(t, assistantartifact.StatusReady, artifact.Status)
	assert.Equal(t, "Revenue by customer", artifact.Title)
	assert.Equal(t, "call_preview_report", artifact.SourceToolCallID)
	assert.Len(t, artifact.Payload["rows"], 2)
	assert.Equal(t, false, artifact.Payload["truncated"])
}

// A preview wider than the store allows is cut to what fits and says so,
// rather than being dropped or stored whole.
func TestArtifactFromObservation_PreviewIsBoundedByThePayloadLimit(t *testing.T) {
	t.Parallel()

	cell := strings.Repeat("x", 4096)
	rows := make([]any, 0, 200)
	for range 200 {
		rows = append(rows, map[string]any{"Notes": cell})
	}
	artifact := artifactFromObservation(observation("preview_report", map[string]any{
		"dataset": "shipments",
		"rows":    rows,
	}))

	require.NotNil(t, artifact)
	assert.Less(t, payloadSize(artifact.Payload), assistantartifact.MaxPayloadBytes)
	assert.Equal(t, true, artifact.Payload["truncated"])
	assert.Equal(t, "Preview of shipments", artifact.Title)
}

// A run's card follows the run: pending until it finishes, failed when it
// did not succeed, ready when there is something to download.
func TestArtifactFromObservation_RunStatusFollowsTheRun(t *testing.T) {
	t.Parallel()

	started := artifactFromObservation(observation("run_report", map[string]any{
		"runId": "rrun_1", "reportKey": "revenue-by-customer", "status": "queued", "finished": false,
	}))
	require.NotNil(t, started)
	assert.Equal(t, assistantartifact.KindReportRun, started.Kind)
	assert.Equal(t, assistantartifact.StatusPending, started.Status)
	assert.Equal(t, "Revenue by customer", started.Title)

	failed := artifactFromObservation(observation("get_report_run", map[string]any{
		"runId": "rrun_1", "reportName": "Revenue", "status": "failed", "finished": true,
	}))
	require.NotNil(t, failed)
	assert.Equal(t, assistantartifact.StatusFailed, failed.Status)

	done := artifactFromObservation(observation("get_report_run", map[string]any{
		"runId": "rrun_1", "reportName": "Revenue", "status": "succeeded", "finished": true,
	}))
	require.NotNil(t, done)
	assert.Equal(t, assistantartifact.StatusReady, done.Status)
}

// A get tool's record is a card named after the record; a get that returns
// a view rather than a record, or nothing, or failed, leaves no card.
func TestArtifactFromObservation_RecordsBecomeCardsAndViewsDoNot(t *testing.T) {
	t.Parallel()

	card := artifactFromObservation(observation("get_shipment", map[string]any{
		"id": "shp_1", "proNumber": "PRO-778", "status": "InTransit",
	}))
	require.NotNil(t, card)
	assert.Equal(t, assistantartifact.KindEntityCard, card.Kind)
	assert.Equal(t, "Shipment PRO-778", card.Title)
	assert.Equal(t, "shipment", card.Payload["entity"])

	person := artifactFromObservation(observation("get_worker", map[string]any{
		"id": "wrk_1", "firstName": "Ada", "lastName": "Lovelace",
	}))
	require.NotNil(t, person)
	assert.Equal(t, "Worker Ada Lovelace", person.Title)

	assert.Nil(t, artifactFromObservation(observation("get_dispatch_board", map[string]any{
		"moves": []any{},
	})), "a board is not a record")
	assert.Nil(t, artifactFromObservation(observation("list_shipments", map[string]any{"id": "x"})))
	assert.Nil(t, artifactFromObservation(serviceports.ToolObservation{
		Call:   serviceports.ToolCall{ID: "call_x", Name: "get_shipment"},
		Failed: true,
	}))
	assert.Nil(t, artifactFromObservation(observation("get_shipment", nil)))
}

// A message proposal reads as a draft: its subject names it, its body is
// what would go, and it points at the proposal that carries the decision.
// A dispatch write is decided from its card and makes no draft.
func TestDraftArtifact_ViewsAnOutboundMessageOverItsProposal(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:              pulid.MustNew("ap_"),
		RunID:           pulid.MustNew("ar_"),
		SourceMessageID: pulid.MustNew("msg_"),
		ToolName:        "email_customer",
		ToolParams: map[string]any{
			"shipmentId": "shp_1",
			"subject":    "Your delivery is running late",
			"body":       "The truck is delayed by two hours.",
		},
		Rationale: "The customer asked for updates.",
		Status:    agent.ProposalStatusPending,
	}

	draft := draftArtifact(proposal)
	require.NotNil(t, draft)
	assert.Equal(t, assistantartifact.KindEmailDraft, draft.Kind)
	assert.Equal(t, assistantartifact.StatusPending, draft.Status)
	assert.Equal(t, "Your delivery is running late", draft.Title)
	assert.Equal(t, "The truck is delayed by two hours.", draft.Payload["body"])
	assert.Equal(t, proposal.ID, draft.ProposalID)
	assert.Equal(t, proposal.SourceMessageID, draft.MessageID)

	notice := draftArtifact(&agent.AgentProposal{
		ID:         pulid.MustNew("ap_"),
		ToolName:   "send_detention_notice",
		ToolParams: map[string]any{"occurrenceId": "dto_1"},
	})
	require.NotNil(t, notice)
	assert.Equal(t, "Detention notice", notice.Title)

	assert.Nil(t, draftArtifact(&agent.AgentProposal{ToolName: "assign_move"}))
}

func TestPlanArtifact_ListsItsStepsInOrder(t *testing.T) {
	t.Parallel()

	planID := pulid.MustNew("apl_")
	other := pulid.MustNew("apl_")
	plan := &agent.AgentPlan{ID: planID, RunID: pulid.MustNew("ar_"), Title: "Cover the move", StepCount: 2}
	steps := []*agent.AgentProposal{
		{ID: pulid.MustNew("ap_"), PlanID: &planID, PlanStep: 1, ToolName: "assign_move"},
		{ID: pulid.MustNew("ap_"), PlanID: &other, PlanStep: 1, ToolName: "email_customer"},
		{ID: pulid.MustNew("ap_"), PlanID: &planID, PlanStep: 2, ToolName: "add_shipment_comment"},
	}

	artifact := planArtifact(plan, steps)
	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindPlan, artifact.Kind)
	assert.Equal(t, planID, artifact.PlanID)
	listed, ok := artifact.Payload["steps"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, listed, 2)
	assert.Equal(t, "assign_move", listed[0]["toolName"])
	assert.Equal(t, "add_shipment_comment", listed[1]["toolName"])
}

// The recorder keeps what a tool produced as the tool finishes and tells the
// stream, so the pane can open the table while the reply is still arriving;
// once the turn is saved it ties each artifact to the message that made it.
func TestArtifactRecorder_KeepsAndAnnouncesWhatATurnProduced(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	var events []serviceports.StreamEvent
	svc := &Service{logger: zap.NewNop(), artifacts: repo}
	thread := &conversation.Thread{ID: pulid.MustNew("thr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	recorder := svc.newArtifactRecorder(t.Context(), thread, tenant, testActor(), func(event serviceports.StreamEvent) {
		events = append(events, event)
	})

	recorder.observe(observation("get_customer", map[string]any{"id": "cus_1", "name": "Acme"}))
	recorder.observe(observation("list_customers", map[string]any{"rows": []any{}}))

	require.Len(t, repo.upserts, 1)
	assert.Equal(t, thread.ID, repo.upserts[0].ThreadID)
	assert.Equal(t, tenant.OrgID, repo.upserts[0].OrganizationID)
	require.Len(t, events, 1)
	assert.Equal(t, serviceports.AssistantEventArtifact, events[0].Event)
	announced, ok := events[0].Data.(serviceports.AssistantArtifactEvent)
	require.True(t, ok)
	assert.Equal(t, assistantartifact.KindEntityCard, announced.Kind)
	assert.Equal(t, "call_get_customer", announced.SourceToolCallID)

	messageID := pulid.MustNew("msg_")
	recorder.attachMessages(map[string]pulid.ID{"call_get_customer": messageID})
	require.Len(t, repo.upserts, 2)
	assert.Equal(t, messageID, repo.upserts[1].MessageID)

	listed := recorder.artifacts()
	require.Len(t, listed, 1)
	assert.Equal(t, messageID, listed[0].MessageID)
}

// A nil recorder is a turn without a pane: every call is a no-op and the
// runtime gets no observer to call.
func TestArtifactRecorder_NilIsSafe(t *testing.T) {
	t.Parallel()

	var recorder *artifactRecorder
	assert.Nil(t, recorder.observer())
	recorder.fromProposals(nil, nil)
	recorder.attachMessages(nil)
	assert.Nil(t, recorder.artifacts())

	svc := &Service{logger: zap.NewNop()}
	assert.Nil(t, svc.newArtifactRecorder(t.Context(), &conversation.Thread{}, pagination.TenantInfo{}, testActor(), nil))
}

// A draft never holds its own decision state. Listing reads the status off
// the proposal, so a draft sent from AI Control reads as sent in the Desk.
func TestListThreadArtifacts_DraftStatusFollowsTheProposal(t *testing.T) {
	t.Parallel()

	thread := &conversation.Thread{ID: pulid.MustNew("thr_")}
	sent := pulid.MustNew("ap_")
	rejected := pulid.MustNew("ap_")
	repo := &stubArtifactRepo{listed: []*assistantartifact.Artifact{
		{ID: pulid.MustNew("art_"), Kind: assistantartifact.KindEmailDraft, Status: assistantartifact.StatusPending, ProposalID: sent},
		{ID: pulid.MustNew("art_"), Kind: assistantartifact.KindEmailDraft, Status: assistantartifact.StatusPending, ProposalID: rejected},
		{ID: pulid.MustNew("art_"), Kind: assistantartifact.KindEntityCard, Status: assistantartifact.StatusReady},
	}}
	svc := &Service{
		logger:        zap.NewNop(),
		artifacts:     repo,
		conversations: &stubConversationRepo{thread: thread},
		proposals: &stubProposalRepo{byThread: []*agent.AgentProposal{
			{ID: sent, Status: agent.ProposalStatusExecuted},
			{ID: rejected, Status: agent.ProposalStatusRejected},
		}},
	}

	listed, err := svc.ListThreadArtifacts(t.Context(), repositories.GetThreadRequest{ID: thread.ID})
	require.NoError(t, err)
	require.Len(t, listed, 3)
	assert.Equal(t, assistantartifact.StatusSent, listed[0].Status)
	assert.Equal(t, assistantartifact.StatusFailed, listed[1].Status)
	assert.Equal(t, assistantartifact.StatusReady, listed[2].Status)
}

/*
A list or a search is the turn that most often earns a table, and for a long
while it was the one turn that produced nothing: only a report preview, a
report run and a single-record get were mapped. So the pane promised a table
beside the conversation and then sat empty through "how many shipments are in
transit", which is the question people actually ask.
*/
func listResult(count int, items ...any) map[string]any {
	return map[string]any{
		"count":       float64(count),
		"searchedFor": []any{"status is InTransit"},
		"columns":     []any{"proNumber", "customer", "status"},
		"items":       items,
	}
}

func TestArtifactFromObservation_AListBecomesATable(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("list_shipments", listResult(2,
		map[string]any{"proNumber": "P1", "customer": "Acme", "status": "InTransit"},
		map[string]any{"proNumber": "P2", "customer": "Globex", "status": "InTransit"},
	)))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindTableView, artifact.Kind)
	assert.Equal(t, assistantartifact.StatusReady, artifact.Status)
	assert.Equal(t, "Shipments", artifact.Title)
	assert.Equal(t, "shipments", artifact.Payload["entity"])
	assert.Equal(t, "list_shipments", artifact.Payload["tool"])
	assert.Equal(t, []string{"proNumber", "customer", "status"}, artifact.Payload["columns"])
	assert.Len(t, artifact.Payload["rows"], 2)
	// The terms that were applied, so the table says what it is a table of.
	assert.Equal(t, []string{"status is InTransit"}, artifact.Payload["searchedFor"])
}

func TestArtifactFromObservation_ASearchBecomesATableToo(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("search_workers", listResult(1,
		map[string]any{"proNumber": "irrelevant"},
	)))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindTableView, artifact.Kind)
	assert.Equal(t, "Workers", artifact.Title)
	assert.Equal(t, "workers", artifact.Payload["entity"])
}

// The guard is the shape, not the name. "list" is a common enough verb that a
// tool named for it can return something that is not a table, and a pane that
// drew one anyway would be drawing a table of nothing.
func TestArtifactFromObservation_AListWithoutRowsOrColumnsIsNotATable(t *testing.T) {
	t.Parallel()

	noColumns := listResult(1, map[string]any{"proNumber": "P1"})
	delete(noColumns, "columns")
	assert.Nil(t, artifactFromObservation(observation("list_reports", noColumns)))

	assert.Nil(t, artifactFromObservation(observation("list_shipments", listResult(0))))
	assert.Nil(t, artifactFromObservation(observation("list_shipments", map[string]any{
		"count": float64(3),
		"items": "not rows",
	})))
}

// Nothing matched is a sentence, not a table with no rows in it.
func TestArtifactFromObservation_AnEmptyListLeavesThePaneAlone(t *testing.T) {
	t.Parallel()

	assert.Nil(t, artifactFromObservation(observation("list_shipments", map[string]any{
		"count":       float64(0),
		"searchedFor": []any{"status is InTransit"},
		"items":       []any{},
		"note":        "No shipments matched status is InTransit.",
	})))
}

// A long list is cut to fit rather than dropped: half a table beside the
// conversation beats the pane staying empty because the answer was big.
func TestArtifactFromObservation_ALongListIsCutToFit(t *testing.T) {
	t.Parallel()

	rows := make([]any, 0, 400)
	for i := range 400 {
		rows = append(rows, map[string]any{
			"proNumber": strings.Repeat("P", 400),
			"customer":  strings.Repeat("C", 400),
			"status":    "InTransit",
			"index":     float64(i),
		})
	}

	artifact := artifactFromObservation(observation("list_shipments", listResult(400, rows...)))

	require.NotNil(t, artifact)
	kept, ok := artifact.Payload["rows"].([]any)
	require.True(t, ok)
	assert.NotEmpty(t, kept)
	assert.Less(t, len(kept), 400)
	// The count stays what the search found, so the footer does not report the
	// truncation as the answer.
	assert.Equal(t, float64(400), artifact.Payload["rowCount"])
}

/*
A described view and a rate explanation are both answers a person works from
rather than reads, so both open in the pane.

The view is a link to the live table, not a snapshot: rows pasted into a
conversation are not sortable, not exportable, not re-checked against
permissions, and wrong by the time anybody reads them.

The rate is a ledger, because a price read aloud is a wall of figures nobody
can check against an invoice.
*/
func TestArtifactFromObservation_ADescribedViewOpensTheTable(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("compose_table_view", map[string]any{
		"entity":      "shipments",
		"path":        "/shipments?fieldFilters=%5B%5D",
		"explanation": "shipments where status equals InTransit",
		"terms":       []any{"status equals InTransit"},
		"filterCount": float64(1),
	}))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindTableView, artifact.Kind)
	assert.Equal(t, "Shipments", artifact.Title)
	assert.Equal(t, "/shipments?fieldFilters=%5B%5D", artifact.Payload["path"])
	assert.Equal(t, []string{"status equals InTransit"}, artifact.Payload["terms"])
}

// A view with no link is not a view, whatever else the result carries.
func TestArtifactFromObservation_ADescribedViewWithNoLinkIsNotOne(t *testing.T) {
	t.Parallel()

	assert.Nil(t, artifactFromObservation(observation("compose_table_view", map[string]any{
		"entity":      "shipments",
		"explanation": "shipments where status equals InTransit",
	})))
}

func TestArtifactFromObservation_ARateBecomesALedger(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("explain_rate", map[string]any{
		"shipmentId": "shp_1",
		"side":       "Customer",
		"components": []any{
			map[string]any{"label": "Linehaul", "basis": "1,240.0 mi @ $2.15/mi"},
		},
		"totals":   map[string]any{"total": "2800.00"},
		"warnings": []any{"fuel index is 6 days stale"},
	}))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindRateExplanation, artifact.Kind)
	assert.Equal(t, "Rate breakdown", artifact.Title)
	assert.Len(t, artifact.Payload["components"], 1)
	assert.Equal(t, []string{"fuel index is 6 days stale"}, artifact.Payload["warnings"])
}

// An unrated shipment has no price to break down. A ledger of zeroes would
// read as a price of zero, which is a number nobody set.
func TestArtifactFromObservation_AnUnratedShipmentGetsNoLedger(t *testing.T) {
	t.Parallel()

	assert.Nil(t, artifactFromObservation(observation("explain_rate", map[string]any{
		"shipmentId": "shp_1",
		"note":       "This shipment has no rating on record for that side.",
		"totals":     map[string]any{"total": "0"},
	})))

	assert.Nil(t, artifactFromObservation(observation("explain_rate", map[string]any{
		"shipmentId": "shp_1",
		"components": []any{},
	})))
}
