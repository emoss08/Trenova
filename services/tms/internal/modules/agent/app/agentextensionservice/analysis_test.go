package agentextensionservice

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSandboxKey = "e2b-live-key"

var testPNG = append([]byte("\x89PNG\r\n\x1a\n"), []byte("chart pixels")...)

type memoryAnalyses struct {
	mu     sync.Mutex
	runs   map[pulid.ID]*agentextension.AnalysisRun
	charts []*agentextension.AnalysisChart
}

func newMemoryAnalyses() *memoryAnalyses {
	return &memoryAnalyses{runs: map[pulid.ID]*agentextension.AnalysisRun{}}
}

func (m *memoryAnalyses) Create(
	_ context.Context,
	run *agentextension.AnalysisRun,
) (*agentextension.AnalysisRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := *run
	m.runs[run.ID] = &copied

	return run, nil
}

func (m *memoryAnalyses) Get(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*agentextension.AnalysisRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	run, ok := m.runs[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("Analysis not found")
	}
	copied := *run

	return &copied, nil
}

func (m *memoryAnalyses) Update(
	_ context.Context,
	run *agentextension.AnalysisRun,
) (*agentextension.AnalysisRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	run.Version++
	copied := *run
	m.runs[run.ID] = &copied

	return run, nil
}

func (m *memoryAnalyses) List(
	_ context.Context,
	_ *agentextensionports.ListAnalysisRunsRequest,
) ([]*agentextension.AnalysisRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*agentextension.AnalysisRun, 0, len(m.runs))
	for _, run := range m.runs {
		copied := *run
		out = append(out, &copied)
	}

	return out, nil
}

func (m *memoryAnalyses) SaveCharts(_ context.Context, charts []*agentextension.AnalysisChart) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.charts = append(m.charts, charts...)

	return nil
}

func (m *memoryAnalyses) GetChart(
	_ context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
	position int,
) (*agentextension.AnalysisChart, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, chart := range m.charts {
		if chart.RunID == runID && chart.Position == position {
			return chart, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Analysis chart not found")
}

func (m *memoryAnalyses) only(t *testing.T) *agentextension.AnalysisRun {
	t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	require.Len(t, m.runs, 1)
	for _, run := range m.runs {
		copied := *run
		return &copied
	}

	return nil
}

type fakeSandbox struct {
	mu        sync.Mutex
	started   []*agentextensionports.SandboxRequest
	files     []agentextensionports.SandboxFile
	ran       []*agentextensionports.RunCodeRequest
	stopped   []string
	execution *agentextensionports.SandboxExecution
	prelude   *agentextensionports.SandboxExecution
	startErr  error
	writeErr  error
}

func newFakeSandbox() *fakeSandbox {
	return &fakeSandbox{
		execution: &agentextensionports.SandboxExecution{},
		prelude:   &agentextensionports.SandboxExecution{},
	}
}

func (f *fakeSandbox) Type() agentextension.Type { return agentextension.TypeE2B }

func (f *fakeSandbox) Classify(err error) agentextensionports.ProviderFailure {
	return (&fakeWeb{}).Classify(err)
}

func (f *fakeSandbox) Test(
	context.Context,
	map[string]string,
) (*agentextensionports.ConnectorTestResult, error) {
	return &agentextensionports.ConnectorTestResult{Note: "The key works."}, nil
}

func (f *fakeSandbox) StartSandbox(
	_ context.Context,
	req *agentextensionports.SandboxRequest,
) (*agentextensionports.Sandbox, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.started = append(f.started, req)
	if f.startErr != nil {
		return nil, f.startErr
	}

	return &agentextensionports.Sandbox{
		ID:          "sbx1",
		Domain:      "e2b.app",
		AccessToken: "envd-token",
		CPUCount:    2,
		MemoryMB:    512,
	}, nil
}

func (f *fakeSandbox) WriteFiles(
	_ context.Context,
	_ *agentextensionports.Sandbox,
	files []agentextensionports.SandboxFile,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.files = append(f.files, files...)

	return f.writeErr
}

func (f *fakeSandbox) RunCode(
	_ context.Context,
	req *agentextensionports.RunCodeRequest,
) (*agentextensionports.SandboxExecution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.ran = append(f.ran, req)
	if req.Code == analysisPrelude {
		return f.prelude, nil
	}

	return f.execution, nil
}

func (f *fakeSandbox) StopSandbox(_ context.Context, _ map[string]string, sandboxID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.stopped = append(f.stopped, sandboxID)

	return nil
}

func (f *fakeSandbox) CostOf(*agentextensionports.Sandbox, time.Duration) decimal.Decimal {
	return decimal.RequireFromString("0.0003")
}

func (f *fixture) enableAnalysis(t *testing.T) {
	t.Helper()

	f.enableType(t, agentextension.TypeE2B, map[string]string{
		"apiKey":     testSandboxKey,
		"runSeconds": "60",
	})
}

func lanesInput() agentextensionports.AnalysisInput {
	return agentextensionports.AnalysisInput{
		Name:      "lanes",
		Source:    agentextension.AnalysisDatasetReportRun,
		SourceID:  pulid.MustNew("rrun_"),
		Label:     "Margin by lane",
		Extension: "csv",
		Content:   []byte("lane,margin\nDAL-ATL,0.18\n"),
		Rows:      1,
	}
}

func (f *fixture) analysisRequest(inputs ...agentextensionports.AnalysisInput) *agentextensionports.AnalysisRequest {
	return &agentextensionports.AnalysisRequest{
		TenantInfo: f.tenant,
		Title:      "Lane margin",
		Purpose:    "Margin by lane for Q3",
		Code:       "import pandas as pd\nprint(pd.read_csv('/home/user/data/lanes.csv'))",
		Inputs:     inputs,
		Owner:      agentextensionports.AnalysisOwner{ThreadID: pulid.MustNew("athr_")},
	}
}

func TestAnalyzeRunsTheCodeOverTheDataAndStopsTheSandbox(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	f.sandbox.execution = &agentextensionports.SandboxExecution{
		Stdout: "lane  margin\nDAL-ATL 0.18\n",
		Results: []agentextensionports.SandboxResult{
			{Text: "<Figure size 640x480>", PNG: testPNG, Title: "Margin by lane"},
			{Text: "0.18"},
			{PNG: []byte("not a png")},
		},
	}

	outcome, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.NoError(t, err)

	assert.Equal(t, agentextension.AnalysisRunStatusSucceeded, outcome.Status)
	assert.Equal(t, []string{"0.18"}, outcome.Results, "a chart's own text is not a result")
	assert.Equal(t, []agentextensionports.AnalysisChartSummary{
		{Position: 1, Title: "Margin by lane"},
	}, outcome.Charts, "an image that is not a PNG is left out")
	require.Len(t, outcome.Datasets, 1)
	assert.Equal(t, "/home/user/data/lanes.csv", outcome.Datasets[0].Path)

	require.Len(t, f.sandbox.started, 1)
	assert.Equal(t, testSandboxKey, f.sandbox.started[0].Settings[agentextension.ConfigKeyAPIKey])
	assert.Equal(t, outcome.RunID.String(), f.sandbox.started[0].Metadata[analysisMetadataRun])
	require.Len(t, f.sandbox.files, 2)
	assert.Equal(t, agentextensionports.SandboxFile{
		Path:    "/home/user/data/lanes.csv",
		Content: []byte("lane,margin\nDAL-ATL,0.18\n"),
	}, f.sandbox.files[0])
	assert.Equal(t, agentextension.AnalysisConfigPath, f.sandbox.files[1].Path)
	assert.JSONEq(t, `{"timezone":"UTC","datasets":[{"name":"lanes",`+
		`"path":"/home/user/data/lanes.csv","extension":"csv"}]}`,
		string(f.sandbox.files[1].Content))
	require.Len(t, f.sandbox.ran, 2)
	assert.Equal(t, analysisPrelude, f.sandbox.ran[0].Code, "the data is prepared first")
	assert.Equal(t, 60*time.Second, f.sandbox.ran[1].Timeout)
	assert.Equal(t, []string{"sbx1"}, f.sandbox.stopped)

	run := f.analyses.only(t)
	assert.Equal(t, agentextension.AnalysisRunStatusSucceeded, run.Status)
	assert.Equal(t, "Margin by lane for Q3", run.Purpose)
	assert.Equal(t, "Lane margin", run.Title)
	assert.Equal(t, "sbx1", run.VendorSandboxID)
	assert.Equal(t, 1, run.ChartCount)
	assert.True(t, decimal.RequireFromString("0.0003").Equal(run.CostUSD))
	assert.Equal(t, f.tenant.UserID, run.RequestedByID)
	require.Len(t, f.analyses.charts, 1)
	assert.Equal(t, testPNG, f.analyses.charts[0].Image)
	assert.Equal(t, 1, f.usage.requests)
	assert.Zero(t, f.usage.failures)
}

func TestAnalyzeKeepsTheTablesAndFiguresTheCodeShowedInOrder(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	f.sandbox.prelude = &agentextensionports.SandboxExecution{
		Results: []agentextensionports.SandboxResult{{JSON: map[string]any{
			"trenova": "datasets",
			"datasets": []any{map[string]any{
				"name": "lanes",
				"rows": float64(1),
				"columns": []any{
					map[string]any{"name": "Lane", "dtype": "string"},
					map[string]any{"name": "Margin", "dtype": "float64"},
				},
			}},
		}}},
	}
	f.sandbox.execution = &agentextensionports.SandboxExecution{
		Results: []agentextensionports.SandboxResult{
			{JSON: map[string]any{
				"trenova": "figure", "label": "Average margin", "value": 0.18,
				"kind": "percent",
			}},
			{JSON: map[string]any{
				"trenova": "table",
				"title":   "Margin by lane",
				"columns": []any{
					map[string]any{"name": "Lane", "kind": "text"},
					map[string]any{"name": "Revenue", "kind": "money", "currency": "usd"},
				},
				"rows":      []any{[]any{"DAL-ATL", 1200.5}},
				"totalRows": float64(1),
			}},
			{PNG: testPNG, Title: "Margin by lane"},
			{JSON: map[string]any{"trenova": "something else"}},
		},
	}

	outcome, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.NoError(t, err)

	assert.Equal(t, []agentextension.AnalysisFigure{{
		Label: "Average margin", Value: 0.18, Kind: agentextension.AnalysisValuePercent,
	}}, outcome.Figures)
	require.Len(t, outcome.Tables, 1)
	assert.Equal(t, []agentextension.AnalysisTableColumn{
		{Name: "Lane", Kind: agentextension.AnalysisValueText},
		{Name: "Revenue", Kind: agentextension.AnalysisValueMoney, Currency: "USD"},
	}, outcome.Tables[0].Columns)
	assert.Equal(t, [][]any{{"DAL-ATL", 1200.5}}, outcome.Tables[0].Rows)
	assert.Equal(t, []agentextension.AnalysisBlock{
		{Kind: agentextension.AnalysisBlockTable, Index: 0},
		{Kind: agentextension.AnalysisBlockChart, Index: 1},
	}, outcome.Layout, "the table came before the chart")
	assert.Empty(t, outcome.Results, "values shown as JSON are not printed results")
	assert.Equal(t, []agentextension.AnalysisDatasetLoad{{
		Name: "lanes",
		Rows: 1,
		Columns: []agentextension.AnalysisDatasetColumn{
			{Name: "Lane", DType: "string"},
			{Name: "Margin", DType: "float64"},
		},
	}}, outcome.Loaded)

	run := f.analyses.only(t)
	assert.Len(t, run.Tables, 1)
	assert.Len(t, run.Figures, 1)
	assert.Len(t, run.Layout, 2)
}

func TestAnalyzeSendsTheReportColumnsAndTimezoneToThePrelude(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	input := lanesInput()
	input.Columns = []agentextension.AnalysisColumnSpec{{
		ID: "margin", Label: "Margin", Type: "decimal", Kind: agentextension.AnalysisValuePercent,
	}}
	req := f.analysisRequest(input)
	req.Timezone = "America/Chicago"

	_, err := f.svc.Analyze(t.Context(), req)
	require.NoError(t, err)

	require.Len(t, f.sandbox.files, 2)
	assert.JSONEq(t, `{"timezone":"America/Chicago","datasets":[{"name":"lanes",`+
		`"path":"/home/user/data/lanes.csv","extension":"csv","columns":[{"id":"margin",`+
		`"label":"Margin","type":"decimal","kind":"percent"}]}]}`,
		string(f.sandbox.files[1].Content))
}

func TestAnalyzeFailsWithoutBlamingTheCodeWhenThePreludeCannotRun(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	f.sandbox.prelude = &agentextensionports.SandboxExecution{
		Error: &agentextension.AnalysisError{Name: "ImportError", Value: "no pandas"},
	}

	_, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.True(t, IsToolError(err))
	assert.Equal(t, analysisPrepFailure, err.Error())
	require.Len(t, f.sandbox.ran, 1, "the agent's code never runs")
	assert.Equal(t, []string{"sbx1"}, f.sandbox.stopped)

	run := f.analyses.only(t)
	assert.Equal(t, agentextension.AnalysisRunStatusFailed, run.Status)
	assert.Equal(t, analysisPrepFailure, run.Failure)
	assert.Equal(t, 1, f.usage.failures)
}

func TestAnalyzeReportsTheErrorTheCodeRaised(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	f.sandbox.execution = &agentextensionports.SandboxExecution{
		Error: &agentextension.AnalysisError{Name: "KeyError", Value: "'margin'", Traceback: "tb"},
	}

	outcome, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.NoError(t, err, "the code's own error is the agent's to fix, not a failure")
	assert.Equal(t, agentextension.AnalysisRunStatusErrored, outcome.Status)
	require.NotNil(t, outcome.Error)
	assert.Equal(t, "KeyError", outcome.Error.Name)
	assert.Equal(t, "KeyError", f.analyses.only(t).ErrorName)
	assert.Zero(t, f.usage.failures)
}

func TestAnalyzeSaysWhenTheCodeRanOutOfTime(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	f.sandbox.execution = &agentextensionports.SandboxExecution{TimedOut: true}

	outcome, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.NoError(t, err)
	assert.Equal(t, agentextension.AnalysisRunStatusTimedOut, outcome.Status)
	assert.Equal(t, "The code was still running after 60 seconds, so it was stopped.",
		outcome.Failure)
	assert.Equal(t, 1, f.usage.failures)
	assert.Equal(t, []string{"sbx1"}, f.sandbox.stopped)
}

func TestAnalyzeRecordsASandboxThatNeverStarted(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	f.sandbox.startErr = &restx.APIError{StatusCode: http.StatusUnauthorized}

	_, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "rejected the organization's API key")

	run := f.analyses.only(t)
	assert.Equal(t, agentextension.AnalysisRunStatusFailed, run.Status)
	assert.Contains(t, run.Failure, "rejected the organization's API key")
	assert.Empty(t, f.sandbox.stopped, "a sandbox that never started has nothing to stop")
	assert.Equal(t, 1, f.usage.failures)
}

func TestAnalyzeStopsTheSandboxWhenTheDataCannotBeLoaded(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	f.sandbox.writeErr = errors.New("connection reset")

	_, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.Error(t, err)
	assert.Equal(t, []string{"sbx1"}, f.sandbox.stopped)
	assert.Empty(t, f.sandbox.ran)
	assert.Equal(t, agentextension.AnalysisRunStatusFailed, f.analyses.only(t).Status)
}

func TestAnalyzeRefusesDatasetsTheSandboxCouldNotBeGiven(t *testing.T) {
	t.Parallel()

	tooBig := lanesInput()
	tooBig.Content = make([]byte, agentextension.MaxAnalysisDatasetBytes+1)
	parquet := lanesInput()
	parquet.Extension = "parquet"
	renamed := lanesInput()
	renamed.Name = "Lanes"
	many := make([]agentextensionports.AnalysisInput, 0, agentextension.MaxAnalysisDatasets+1)
	for idx := range agentextension.MaxAnalysisDatasets + 1 {
		input := lanesInput()
		input.Name = "d" + strings.Repeat("x", idx)
		many = append(many, input)
	}

	tests := map[string][]agentextensionports.AnalysisInput{
		"too large":   {tooBig},
		"a format":    {parquet},
		"a bad name":  {renamed},
		"a duplicate": {lanesInput(), lanesInput()},
		"too many":    many,
	}
	for name, inputs := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.enableAnalysis(t)
			_, err := f.svc.Analyze(t.Context(), f.analysisRequest(inputs...))
			require.True(t, IsToolError(err), err)
			assert.Empty(t, f.sandbox.started)
			assert.Zero(t, f.usage.requests)
		})
	}
}

func TestAnalyzeNeedsTheExtensionOn(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "Data analysis is not set up")
	assert.Empty(t, f.sandbox.started)
}

func TestAnalysisChartsAreShownOnlyToWhoTheyWereRunFor(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	f.sandbox.execution = &agentextensionports.SandboxExecution{
		Results: []agentextensionports.SandboxResult{{PNG: testPNG}},
	}
	outcome, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.NoError(t, err)

	mine, err := f.svc.AnalysisChart(t.Context(), &agentextensionports.AnalysisChartRequest{
		TenantInfo: f.tenant,
		RunID:      outcome.RunID,
		Position:   1,
	})
	require.NoError(t, err)
	assert.Equal(t, testPNG, mine.Image)

	other := f.tenant
	other.UserID = pulid.MustNew("usr_")
	_, err = f.svc.AnalysisChart(t.Context(), &agentextensionports.AnalysisChartRequest{
		TenantInfo: other,
		RunID:      outcome.RunID,
		Position:   1,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = f.svc.AnalysisChart(t.Context(), &agentextensionports.AnalysisChartRequest{
		TenantInfo:  other,
		RunID:       outcome.RunID,
		Position:    1,
		Administers: true,
	})
	require.NoError(t, err, "someone who oversees agents sees every run's charts")

	_, err = f.svc.AnalysisChart(t.Context(), &agentextensionports.AnalysisChartRequest{
		TenantInfo: f.tenant,
		RunID:      outcome.RunID,
		Position:   2,
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestAnAnalysisIsReadWholeOnlyByWhoItWasRunFor(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableAnalysis(t)
	outcome, err := f.svc.Analyze(t.Context(), f.analysisRequest(lanesInput()))
	require.NoError(t, err)

	mine, err := f.svc.AnalysisRun(t.Context(), &agentextensionports.AnalysisRunRequest{
		TenantInfo: f.tenant,
		RunID:      outcome.RunID,
	})
	require.NoError(t, err)
	assert.Equal(t, "Lane margin", mine.Title)

	other := f.tenant
	other.UserID = pulid.MustNew("usr_")
	_, err = f.svc.AnalysisRun(t.Context(), &agentextensionports.AnalysisRunRequest{
		TenantInfo: other,
		RunID:      outcome.RunID,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = f.svc.AnalysisRun(t.Context(), &agentextensionports.AnalysisRunRequest{
		TenantInfo:  other,
		RunID:       outcome.RunID,
		Administers: true,
	})
	require.NoError(t, err)
}
