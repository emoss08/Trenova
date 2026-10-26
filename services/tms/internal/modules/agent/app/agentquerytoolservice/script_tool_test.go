package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubScriptRunner struct {
	asked   *scriptports.Request
	outcome *scriptports.Outcome
}

func (s *stubScriptRunner) Run(
	_ context.Context,
	req *scriptports.Request,
) (*scriptports.Outcome, error) {
	s.asked = req
	if s.outcome != nil {
		return s.outcome, nil
	}

	return &scriptports.Outcome{Status: scriptports.Succeeded, Output: "done\n"}, nil
}

type stubScriptBroker struct{}

func (stubScriptBroker) Index() []codemode.Read                             { return nil }
func (stubScriptBroker) Bound(*codemode.Limits)                             {}
func (stubScriptBroker) Read(context.Context, codemode.Call) codemode.Reply { return codemode.Reply{} }
func (stubScriptBroker) NoteReply(int, int64)                               {}
func (stubScriptBroker) Summary() codemode.Summary                          { return codemode.Summary{} }

type stubAnalysisRecorder struct {
	recorded *analysis.Run
}

func (s *stubAnalysisRecorder) Record(_ context.Context, run *analysis.Run) (*analysis.Run, error) {
	run.Bound()
	run.ID = pulid.MustNew(analysis.RunIDPrefix)
	s.recorded = run

	return run, nil
}

type stubConversationFiles struct {
	file *documentports.ConversationFile
}

func (s *stubConversationFiles) ReadConversationFile(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.ConversationFile, error) {
	if s.file == nil || s.file.Document.ID != req.ID {
		return nil, errortypes.NewNotFoundError("Document not found")
	}

	return s.file, nil
}

type scriptHarness struct {
	tool     *runScriptTool
	runner   *stubScriptRunner
	reports  *fakeReporting
	files    *stubConversationFiles
	perms    *fakePermissions
	analyses *stubAnalysisRecorder
	params   *agentports.QueryToolParams
}

func newScriptHarness() *scriptHarness {
	h := &scriptHarness{
		runner:   &stubScriptRunner{},
		reports:  &fakeReporting{rows: map[pulid.ID]*export.Envelope{}},
		files:    &stubConversationFiles{},
		perms:    &fakePermissions{allowed: true},
		analyses: &stubAnalysisRecorder{},
		params:   personParams(permissionports.PrincipalTypeUser),
	}
	h.params.ThreadID = pulid.MustNew("athr_")
	h.params.Broker = stubScriptBroker{}
	tool, ok := newRunScriptTool(&runScriptToolParams{
		Runner:      h.runner,
		Reports:     h.reports,
		Files:       h.files,
		Permissions: h.perms,
		Analyses:    h.analyses,
	}).(*runScriptTool)
	if !ok {
		panic("newRunScriptTool must return *runScriptTool")
	}
	h.tool = tool

	return h
}

func (h *scriptHarness) seedRun(t *testing.T, requestedBy pulid.ID) pulid.ID {
	t.Helper()
	run := &report.ReportRun{ID: pulid.MustNew("rrun_"), RequestedByID: requestedBy}
	h.reports.runs = append(h.reports.runs, run)
	margin := testColumn("margin", "Margin", value.Decimal)
	margin.Display = format.Resolve(value.Decimal, format.Hint{Kind: format.Percent}, nil)
	invoiced := testColumn("invoiced", "Invoiced", value.Epoch)
	invoiced.Display.DateStyle = format.DateStyleDate
	h.reports.rows[run.ID] = testEnvelope(t, "Margin by lane",
		[]table.Column{testColumn("lane", "Lane", value.String), margin, invoiced},
		[]any{"DAL-ATL", "0.180000", int64(1789000000)},
		[]any{"HOU-MEM", nil, nil})

	return run.ID
}

func (h *scriptHarness) attach(name, content string, thread, owner pulid.ID) pulid.ID {
	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: name,
		ResourceType: document.ConversationResourceType,
		ResourceID:   thread.String(),
		UploadedByID: owner,
	}
	h.files.file = &documentports.ConversationFile{Document: doc, Data: []byte(content)}

	return doc.ID
}

func (h *scriptHarness) query(t *testing.T, datasets ...map[string]any) (*scriptAnswer, error) {
	t.Helper()

	items := make([]any, 0, len(datasets))
	for _, dataset := range datasets {
		items = append(items, dataset)
	}
	h.params.Params = map[string]any{
		"title":    "Lane margin",
		"purpose":  "Margin by lane for Q3",
		"code":     "show_figure('Lanes', len(lanes))",
		"datasets": items,
	}
	result, err := h.tool.Query(t.Context(), h.params)
	if err != nil {
		return nil, err
	}
	answer, ok := result.(*scriptAnswer)
	require.True(t, ok)

	return answer, nil
}

func shownFigure() analysis.Display {
	var shown analysis.Display
	shown.AddFigure(&analysis.Figure{Label: "Lanes", Value: 2.0})

	return shown
}

func TestRunScriptReadsTheRunsTheCallerRanTypedByTheirColumns(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	h.params.Timezone = "America/Chicago"
	runID := h.seedRun(t, h.params.Actor.UserID)

	answer, err := h.query(t, map[string]any{"name": "lanes", "reportRunId": runID.String()})
	require.NoError(t, err)

	require.NotNil(t, h.runner.asked)
	require.Len(t, h.runner.asked.Datasets, 1)
	dataset := h.runner.asked.Datasets[0]
	assert.Equal(t, "lanes", dataset.Name)
	assert.Equal(t, analysis.DatasetReportRun, dataset.Source)
	assert.Equal(t, runID, dataset.SourceID)
	assert.Equal(t, "Margin by lane", dataset.Label)
	assert.Equal(t, []scriptports.DatasetColumn{
		{Name: "Lane", Kind: analysis.ValueText},
		{Name: "Margin", Kind: analysis.ValuePercent},
		{Name: "Invoiced", Kind: analysis.ValueDate},
	}, dataset.Columns, "each column is headed by its label and keeps its kind")
	assert.Equal(t, [][]any{
		{"DAL-ATL", 0.18, "2026-09-09"},
		{"HOU-MEM", nil, nil},
	}, dataset.Rows, "numbers are numbers and a date is the organization's day")
	assert.Equal(t, permission.ResourceReport.String(), h.perms.captured.Resource)
	assert.Equal(t, permission.OpRead, h.perms.captured.Operation)

	require.Len(t, answer.Datasets, 1)
	assert.Equal(t, 2, answer.Datasets[0].Rows)
	assert.Empty(t, answer.TaintedRecords(), "a report run is the organization's own data")
}

func TestRunScriptRefusesARunSomeoneElseRan(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	runID := h.seedRun(t, pulid.MustNew("usr_"))

	_, err := h.query(t, map[string]any{"name": "lanes", "reportRunId": runID.String()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not one you ran for this person")
	assert.Nil(t, h.runner.asked)
}

func TestRunScriptNeedsPermissionToReadReports(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	h.perms.allowed = false
	runID := h.seedRun(t, h.params.Actor.UserID)

	_, err := h.query(t, map[string]any{"name": "lanes", "reportRunId": runID.String()})
	require.ErrorIs(t, err, errDatasetsReportRead)
	assert.Nil(t, h.runner.asked)
}

func TestRunScriptReadsAFileThePersonAttachedAndCarriesItsTaint(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	docID := h.attach("Carrier Invoices.CSV",
		"carrier,invoiced,billed on\nAcme,\"$1,200.50\",2026-09-01\nPeak,80,2026-09-02\n",
		h.params.ThreadID, h.params.Actor.UserID)

	answer, err := h.query(t, map[string]any{"name": "invoices", "documentId": docID.String()})
	require.NoError(t, err)

	dataset := h.runner.asked.Datasets[0]
	assert.Equal(t, analysis.DatasetAttachment, dataset.Source)
	assert.Equal(t, "Carrier Invoices.CSV", dataset.Label)
	assert.Equal(t, []scriptports.DatasetColumn{
		{Name: "carrier", Kind: analysis.ValueText},
		{Name: "invoiced", Kind: analysis.ValueNumber},
		{Name: "billed on", Kind: analysis.ValueDate},
	}, dataset.Columns)
	assert.Equal(t, [][]any{
		{"Acme", 1200.5, "2026-09-01"},
		{"Peak", 80.0, "2026-09-02"},
	}, dataset.Rows, "a spreadsheet's money reads as a number")
	assert.Equal(t, []agent.RecordRef{{EntityType: agent.TaintEntityDocument, ID: docID.String()}},
		answer.TaintedRecords(), "what the person attached may have been written outside")
}

func TestRunScriptReadsATextFileAsItsText(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	docID := h.attach("notes.txt", "lane DAL-ATL is late twice a week",
		h.params.ThreadID, h.params.Actor.UserID)

	_, err := h.query(t, map[string]any{"name": "notes", "documentId": docID.String()})
	require.NoError(t, err)

	dataset := h.runner.asked.Datasets[0]
	require.NotNil(t, dataset.Text)
	assert.Equal(t, "lane DAL-ATL is late twice a week", *dataset.Text)
	assert.Empty(t, dataset.Rows)
}

func TestRunScriptRefusesAFileItCannotRead(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	docID := h.attach("rate-con.pdf", "%PDF-1.7", h.params.ThreadID, h.params.Actor.UserID)

	_, err := h.query(t, map[string]any{"name": "ratecon", "documentId": docID.String()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use a CSV, XLSX or text file")
	assert.Nil(t, h.runner.asked)
}

func TestRunScriptRefusesAFileFromAnotherConversation(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	docID := h.attach("invoices.csv", "carrier\nAcme\n", pulid.MustNew("athr_"),
		h.params.Actor.UserID)

	_, err := h.query(t, map[string]any{"name": "invoices", "documentId": docID.String()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was not attached to this conversation")
	assert.Nil(t, h.runner.asked)
}

func TestRunScriptRefusesAFileSomeoneElseAttached(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	docID := h.attach("invoices.csv", "carrier\nAcme\n", h.params.ThreadID, pulid.MustNew("usr_"))

	_, err := h.query(t, map[string]any{"name": "invoices", "documentId": docID.String()})
	require.Error(t, err)
	assert.Nil(t, h.runner.asked)
}

func TestRunScriptNeedsADatasetNamedOnceWithOneSource(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	runID := h.seedRun(t, h.params.Actor.UserID)

	for name, datasets := range map[string][]map[string]any{
		"no source": {{"name": "lanes"}},
		"two sources": {{
			"name":        "lanes",
			"reportRunId": runID.String(),
			"documentId":  pulid.MustNew("doc_").String(),
		}},
		"not a python name":   {{"name": "Lanes Q3", "reportRunId": runID.String()}},
		"one of the script's": {{"name": "trenova", "reportRunId": runID.String()}},
		"the same name twice": {
			{"name": "lanes", "reportRunId": runID.String()},
			{"name": "lanes", "reportRunId": runID.String()},
		},
	} {
		_, err := h.query(t, datasets...)
		require.Error(t, err, name)
	}
	assert.Nil(t, h.runner.asked)
}

func TestRunScriptKeepsWhatAScriptShowedAsAnAnalysis(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	runID := h.seedRun(t, h.params.Actor.UserID)
	shown := shownFigure()
	rows := make([][]any, 0, 60)
	for idx := range 60 {
		rows = append(rows, []any{float64(idx)})
	}
	shown.AddTable(&analysis.Table{
		Title:   "Every lane",
		Columns: []analysis.TableColumn{{Name: "n", Kind: analysis.ValueCount}},
		Rows:    rows,
	})
	h.runner.outcome = &scriptports.Outcome{
		Status:   scriptports.Succeeded,
		Output:   "2 lanes\n",
		Shown:    shown,
		NotShown: 1,
	}

	answer, err := h.query(t, map[string]any{"name": "lanes", "reportRunId": runID.String()})
	require.NoError(t, err)

	run := h.analyses.recorded
	require.NotNil(t, run)
	assert.Equal(t, analysis.StatusSucceeded, run.Status)
	assert.Equal(t, "Lane margin", run.Title)
	assert.Equal(t, "Margin by lane for Q3", run.Purpose)
	assert.Equal(t, h.params.Actor.UserID, run.RequestedByID)
	assert.Equal(t, h.params.ThreadID, run.ThreadID)
	assert.Equal(t, []analysis.Dataset{{
		Name:     "lanes",
		Source:   analysis.DatasetReportRun,
		SourceID: runID,
		Label:    "Margin by lane",
		Rows:     2,
	}}, run.Datasets, "the run names what it read, not its rows")
	require.Len(t, run.Tables, 1)
	assert.Len(t, run.Tables[0].Rows, 60, "the card reads every row the script showed")
	assert.GreaterOrEqual(t, run.FinishedAt, run.StartedAt)

	assert.Equal(t, run.ID.String(), answer.AnalysisRunID)
	require.Len(t, answer.Tables, 1)
	assert.Len(t, answer.Tables[0].Rows, maxAnswerTableRows, "the agent reads the first rows")
	assert.Len(t, answer.Figures, 1)
	assert.Equal(t, 1, answer.NotShown)
	assert.Contains(t, answer.Note, scriptShownNote)
	assert.Contains(t, answer.Note, scriptNotShownNote)
}

func TestRunScriptThatOnlyPrintedIsNotKept(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()

	answer, err := h.query(t)
	require.NoError(t, err)

	assert.Nil(t, h.analyses.recorded, "a script that showed nothing is the agent looking")
	assert.Empty(t, answer.AnalysisRunID)
	assert.Contains(t, answer.Note, scriptPlainNote)
}

func TestRunScriptKeepsTheErrorOfAScriptThatShowedAndRaised(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	h.runner.outcome = &scriptports.Outcome{
		Status: scriptports.Errored,
		Shown:  shownFigure(),
		Error: &codemode.ScriptError{
			Name:      "KeyError",
			Value:     "'margin'",
			Traceback: "File \"script.py\", line 3",
		},
	}

	answer, err := h.query(t)
	require.NoError(t, err)

	run := h.analyses.recorded
	require.NotNil(t, run)
	assert.Equal(t, analysis.StatusErrored, run.Status)
	assert.Equal(t, "KeyError", run.ErrorName)
	assert.Equal(t, "'margin'", run.ErrorValue)
	assert.NotContains(t, answer.Note, scriptPlainNote)
}

func TestRunScriptDeclaresWhatItReads(t *testing.T) {
	t.Parallel()

	policy := newScriptHarness().tool.Policy()
	assert.Equal(t, permission.ResourceCodeMode, policy.Resource)
	assert.Equal(t, agent.ExternalReadMarked, policy.ReadsExternal)
	assert.Contains(t, policy.Sources, agent.TaintSourceAttachment,
		"an attached file it reads is outside text")
}
