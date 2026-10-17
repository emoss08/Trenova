package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAnalyzer struct {
	asked   *agentextensionports.AnalysisRequest
	outcome *agentextensionports.AnalysisOutcome
}

func (s *stubAnalyzer) Analyze(
	_ context.Context,
	req *agentextensionports.AnalysisRequest,
) (*agentextensionports.AnalysisOutcome, error) {
	s.asked = req
	if s.outcome != nil {
		return s.outcome, nil
	}

	datasets := make([]agentextension.AnalysisDataset, 0, len(req.Inputs))
	for idx := range req.Inputs {
		input := &req.Inputs[idx]
		datasets = append(datasets, agentextension.AnalysisDataset{
			Name:   input.Name,
			Source: input.Source,
			Label:  input.Label,
			Path:   agentextension.DatasetPath(input.Name, input.Extension),
			Rows:   input.Rows,
		})
	}

	return &agentextensionports.AnalysisOutcome{
		RunID:    pulid.MustNew(agentextension.AnalysisRunIDPrefix),
		Status:   agentextension.AnalysisRunStatusSucceeded,
		Datasets: datasets,
		Stdout:   "margin 0.18\n",
		Results:  []string{"0.18"},
		Charts:   []agentextensionports.AnalysisChartSummary{{Position: 1, Title: "Margin by lane"}},
	}, nil
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

type analysisHarness struct {
	tool     agentports.AgentQueryTool
	reports  *fakeReporting
	files    *stubConversationFiles
	analyzer *stubAnalyzer
	perms    *fakePermissions
	params   *agentports.QueryToolParams
}

func newAnalysisHarness() *analysisHarness {
	h := &analysisHarness{
		reports:  &fakeReporting{rows: map[pulid.ID]*reportrows.Envelope{}},
		files:    &stubConversationFiles{},
		analyzer: &stubAnalyzer{},
		perms:    &fakePermissions{allowed: true},
		params:   personParams(permissionports.PrincipalTypeUser),
	}
	h.params.ThreadID = pulid.MustNew("athr_")
	h.tool = newAnalyzeDataTool(h.reports, h.files, h.analyzer, h.perms)

	return h
}

func (h *analysisHarness) seedRun(requestedBy pulid.ID) pulid.ID {
	run := &report.ReportRun{ID: pulid.MustNew("rrun_"), RequestedByID: requestedBy}
	h.reports.runs = append(h.reports.runs, run)
	h.reports.rows[run.ID] = &reportrows.Envelope{
		Meta:   reportrows.Meta{Title: "Margin by lane"},
		Schema: []reportrows.Column{{ID: "lane"}, {ID: "margin"}},
		Rows:   [][]any{{"DAL-ATL", "0.180000"}, {"HOU-MEM", nil}},
	}

	return run.ID
}

func (h *analysisHarness) attach(name string, thread, owner pulid.ID) pulid.ID {
	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: name,
		ResourceType: document.ConversationResourceType,
		ResourceID:   thread.String(),
		UploadedByID: owner,
	}
	h.files.file = &documentports.ConversationFile{Document: doc, Data: []byte("carrier,invoiced\n")}

	return doc.ID
}

func (h *analysisHarness) query(t *testing.T, datasets ...map[string]any) (*analysisAnswer, error) {
	t.Helper()

	items := make([]any, 0, len(datasets))
	for _, dataset := range datasets {
		items = append(items, dataset)
	}
	h.params.Params = map[string]any{
		"purpose":  "Margin by lane for Q3",
		"code":     "import pandas as pd\nprint(pd.read_csv('/home/user/data/lanes.csv'))",
		"datasets": items,
	}
	result, err := h.tool.Query(t.Context(), h.params)
	if err != nil {
		return nil, err
	}
	answer, ok := result.(*analysisAnswer)
	require.True(t, ok)

	return answer, nil
}

func TestAnalyzeDataReadsTheRunsTheCallerRanAsRawCSV(t *testing.T) {
	t.Parallel()

	h := newAnalysisHarness()
	runID := h.seedRun(h.params.Actor.UserID)

	answer, err := h.query(t, map[string]any{"name": "lanes", "reportRunId": runID.String()})
	require.NoError(t, err)

	require.NotNil(t, h.analyzer.asked)
	require.Len(t, h.analyzer.asked.Inputs, 1)
	input := h.analyzer.asked.Inputs[0]
	assert.Equal(t, "lanes", input.Name)
	assert.Equal(t, "csv", input.Extension)
	assert.Equal(t, "Margin by lane", input.Label)
	assert.Equal(t, 2, input.Rows)
	assert.Equal(t, "lane,margin\nDAL-ATL,0.180000\nHOU-MEM,\n", string(input.Content))
	assert.Equal(t, h.params.ThreadID, h.analyzer.asked.Owner.ThreadID)
	assert.Equal(t, permission.ResourceReport.String(), h.perms.captured.Resource)
	assert.Equal(t, permission.OpRead, h.perms.captured.Operation)

	assert.Equal(t, "Succeeded", answer.Status)
	assert.Equal(t, "Margin by lane for Q3", answer.Purpose)
	assert.Equal(t, "/home/user/data/lanes.csv", answer.Datasets[0].Path)
	assert.Equal(t, "margin 0.18", answer.Output)
	assert.Contains(t, answer.Note, "charts are shown to the person")
	assert.Empty(t, answer.TaintedRecords(), "a report run is the organization's own data")
}

func TestAnalyzeDataRefusesARunSomeoneElseRan(t *testing.T) {
	t.Parallel()

	h := newAnalysisHarness()
	runID := h.seedRun(pulid.MustNew("usr_"))

	_, err := h.query(t, map[string]any{"name": "lanes", "reportRunId": runID.String()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not one you ran for this person")
	assert.Nil(t, h.analyzer.asked)
}

func TestAnalyzeDataNeedsPermissionToReadReports(t *testing.T) {
	t.Parallel()

	h := newAnalysisHarness()
	h.perms.allowed = false
	runID := h.seedRun(h.params.Actor.UserID)

	_, err := h.query(t, map[string]any{"name": "lanes", "reportRunId": runID.String()})
	require.ErrorIs(t, err, errAnalysisReportRead)
	assert.Nil(t, h.analyzer.asked)
}

func TestAnalyzeDataReadsAFileThePersonAttachedAndCarriesItsTaint(t *testing.T) {
	t.Parallel()

	h := newAnalysisHarness()
	docID := h.attach("Carrier Invoices.XLSX", h.params.ThreadID, h.params.Actor.UserID)

	answer, err := h.query(t, map[string]any{"name": "invoices", "documentId": docID.String()})
	require.NoError(t, err)

	input := h.analyzer.asked.Inputs[0]
	assert.Equal(t, "xlsx", input.Extension)
	assert.Equal(t, agentextension.AnalysisDatasetAttachment, input.Source)
	assert.Equal(t, "Carrier Invoices.XLSX", input.Label)
	assert.Equal(t, []agent.RecordRef{{EntityType: agent.TaintEntityDocument, ID: docID.String()}},
		answer.TaintedRecords())
}

func TestAnalyzeDataRefusesAFileFromAnotherConversation(t *testing.T) {
	t.Parallel()

	h := newAnalysisHarness()
	docID := h.attach("invoices.csv", pulid.MustNew("athr_"), h.params.Actor.UserID)

	_, err := h.query(t, map[string]any{"name": "invoices", "documentId": docID.String()})
	require.Error(t, err)
	assert.Nil(t, h.analyzer.asked)
}

func TestAnalyzeDataNeedsExactlyOneSourcePerDataset(t *testing.T) {
	t.Parallel()

	h := newAnalysisHarness()
	runID := h.seedRun(h.params.Actor.UserID)

	_, err := h.query(t, map[string]any{"name": "lanes"})
	require.Error(t, err)
	_, err = h.query(t, map[string]any{
		"name":        "lanes",
		"reportRunId": runID.String(),
		"documentId":  pulid.MustNew("doc_").String(),
	})
	require.Error(t, err)
	_, err = h.query(t, map[string]any{"name": "Lanes Q3", "reportRunId": runID.String()})
	require.Error(t, err)
	assert.Nil(t, h.analyzer.asked)
}

func TestAnalyzeDataTellsTheAgentToFixCodeThatRaised(t *testing.T) {
	t.Parallel()

	h := newAnalysisHarness()
	h.analyzer.outcome = &agentextensionports.AnalysisOutcome{
		RunID:  pulid.MustNew(agentextension.AnalysisRunIDPrefix),
		Status: agentextension.AnalysisRunStatusErrored,
		Error: &agentextension.AnalysisError{
			Name:      "KeyError",
			Value:     "'margin'",
			Traceback: strings.Repeat("frame\n", 2000),
		},
	}

	answer, err := h.query(t)
	require.NoError(t, err)
	require.NotNil(t, answer.Error)
	assert.Equal(t, "KeyError", answer.Error.Name)
	assert.LessOrEqual(t, len([]rune(answer.Error.Traceback)), maxAnalysisTracebackRunes)
	assert.Equal(t, analysisErrorNote, answer.Note)
}

func TestAnalyzeDataSaysWhyAnAnalysisStopped(t *testing.T) {
	t.Parallel()

	h := newAnalysisHarness()
	h.analyzer.outcome = &agentextensionports.AnalysisOutcome{
		RunID:   pulid.MustNew(agentextension.AnalysisRunIDPrefix),
		Status:  agentextension.AnalysisRunStatusTimedOut,
		Failure: "The code was still running after 90 seconds, so it was stopped.",
	}

	answer, err := h.query(t)
	require.NoError(t, err)
	assert.Equal(t, "The code was still running after 90 seconds, so it was stopped.", answer.Failure)
	assert.True(t, strings.HasPrefix(answer.Note, answer.Failure))
}

func TestAnalysisOutputKeepsTheEndOfALongPrint(t *testing.T) {
	t.Parallel()

	output := analysisOutput(strings.Repeat("a", maxAnalysisOutputRunes)+"SUMMARY", "warning: x")
	assert.LessOrEqual(t, len([]rune(output)), maxAnalysisOutputRunes)
	assert.True(t, strings.HasSuffix(output, "[stderr]\nwarning: x"))
	assert.True(t, strings.HasPrefix(output, "…"))
}

func TestAnalyzeDataDeclaresWhatItReads(t *testing.T) {
	t.Parallel()

	policy := newAnalysisHarness().tool.Policy()
	assert.Equal(t, permission.ResourceDataAnalysis, policy.Resource)
	assert.Equal(t, agent.ExternalReadMarked, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceAttachment, policy.Source)
	assert.False(t, policy.CapsLaterWrites)
}
