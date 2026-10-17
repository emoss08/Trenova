package agentextensionservice

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	analysisMetadataRun  = "trenovaAnalysisRun"
	analysisStopTimeout  = 20 * time.Second
	maxAnalysisPurpose   = 500
	analysisRunListLimit = 200
)

var (
	_ agentextensionports.DataAnalyzer = (*Service)(nil)
	_ agentextensionports.AnalysisRuns = (*Service)(nil)

	pngSignature = []byte("\x89PNG\r\n\x1a\n")

	// analysisExtensions are the formats a dataset may be written as: what
	// pandas reads without anything the sandbox would have to fetch.
	analysisExtensions = []string{"csv", "xlsx", "json", "txt"}
)

type analysisCall struct {
	*extensionSettings
	settings agentextension.E2BSettings
	provider agentextensionports.SandboxProvider
}

func (s *Service) dataAnalysis(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*analysisCall, error) {
	extension, err := s.serving(ctx, tenantInfo, agentextension.CapabilityDataAnalysis)
	if err != nil {
		return nil, err
	}

	provider, ok := s.sandboxes[extension.typ]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"the {0} extension cannot run code", extension.typ.String(),
		)
	}

	settings, err := agentextension.ParseE2BSettings(extension.values)
	if err != nil {
		s.l.Error("stored data analysis settings are invalid",
			zap.String("type", extension.typ.String()), zap.Error(err))
		return nil, &ToolError{Message: "The " + extension.def.title() + invalidSettingsSuffix}
	}

	return &analysisCall{extensionSettings: extension, settings: settings, provider: provider}, nil
}

// checkInputs refuses datasets the sandbox could not be given: too many, too
// large, named so the code could not open them, or in a format it would have
// to fetch something to read.
func checkInputs(inputs []agentextensionports.AnalysisInput) error {
	if len(inputs) > agentextension.MaxAnalysisDatasets {
		return &ToolError{Message: fmt.Sprintf(
			"An analysis reads at most %d datasets. Run it on fewer, or combine them in the report first.",
			agentextension.MaxAnalysisDatasets,
		)}
	}

	seen := make(map[string]struct{}, len(inputs))
	total := 0
	for idx := range inputs {
		input := &inputs[idx]
		if err := agentextension.CheckDatasetName(input.Name); err != nil {
			return &ToolError{Message: stringutils.CapitalizeFirst(err.Error()) + "."}
		}
		if _, dup := seen[input.Name]; dup {
			return &ToolError{Message: "Two datasets are named " + input.Name + "; give each its own name."}
		}
		seen[input.Name] = struct{}{}
		if !slices.Contains(analysisExtensions, input.Extension) {
			return &ToolError{Message: "Dataset " + input.Name + " is a ." + input.Extension +
				" file, which an analysis cannot read. Use a CSV, XLSX, JSON or text file."}
		}
		if len(input.Content) > agentextension.MaxAnalysisDatasetBytes {
			return &ToolError{Message: "Dataset " + input.Name + " is larger than 20 MB. " +
				"Narrow the report's filters or columns and run it again."}
		}
		total += len(input.Content)
	}
	if total > agentextension.MaxAnalysisInputBytes {
		return &ToolError{Message: "The datasets together are larger than 40 MB. " +
			"Narrow them, or analyze them one at a time."}
	}

	return nil
}

func datasetsOf(inputs []agentextensionports.AnalysisInput) []agentextension.AnalysisDataset {
	datasets := make([]agentextension.AnalysisDataset, 0, len(inputs))
	for idx := range inputs {
		input := &inputs[idx]
		datasets = append(datasets, agentextension.AnalysisDataset{
			Name:     input.Name,
			Source:   input.Source,
			SourceID: input.SourceID,
			Label:    stringutils.TruncateRunes(strings.TrimSpace(input.Label), 200),
			Path:     agentextension.DatasetPath(input.Name, input.Extension),
			Rows:     input.Rows,
			Bytes:    int64(len(input.Content)),
		})
	}

	return datasets
}

// Analyze runs the agent's code over the datasets in a sandbox that cannot
// reach the network, and stops the sandbox whatever happens. The run is
// recorded before the vendor is called, so a sandbox Trenova started is never
// one it has no record of.
func (s *Service) Analyze(
	ctx context.Context,
	req *agentextensionports.AnalysisRequest,
) (*agentextensionports.AnalysisOutcome, error) {
	if err := agentextension.CheckAnalysisCode(req.Code); err != nil {
		return nil, &ToolError{Message: stringutils.CapitalizeFirst(err.Error()) + "."}
	}
	if err := checkInputs(req.Inputs); err != nil {
		return nil, err
	}

	call, err := s.dataAnalysis(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	metered := meteredCall{
		tenant: req.TenantInfo,
		typ:    call.typ,
		limit:  call.settings.DailyRequestLimit,
	}
	day, err := s.reserve(ctx, metered)
	if err != nil {
		return nil, err
	}

	run, err := s.analyses.Create(ctx, s.newAnalysisRun(req, call.typ))
	if err != nil {
		s.settle(ctx, metered, settledCall{day: day, failed: true})
		return nil, err
	}

	analysis := &analysisAttempt{
		call:    call,
		metered: metered,
		day:     day,
		run:     run,
	}

	return s.runAnalysis(ctx, analysis, req)
}

type analysisAttempt struct {
	call    *analysisCall
	metered meteredCall
	day     int
	run     *agentextension.AnalysisRun
	sandbox *agentextensionports.Sandbox
	started time.Time
}

func (s *Service) newAnalysisRun(
	req *agentextensionports.AnalysisRequest,
	typ agentextension.Type,
) *agentextension.AnalysisRun {
	run := &agentextension.AnalysisRun{
		ID:                pulid.MustNew(agentextension.AnalysisRunIDPrefix),
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		ExtensionType:     typ,
		Status:            agentextension.AnalysisRunStatusRunning,
		Purpose:           stringutils.TruncateRunes(strings.TrimSpace(req.Purpose), maxAnalysisPurpose),
		Code:              req.Code,
		Datasets:          datasetsOf(req.Inputs),
		Results:           []string{},
		AgentDefinitionID: req.Owner.DefinitionID,
		AgentRunID:        req.Owner.RunID,
		ThreadID:          req.Owner.ThreadID,
		RequestedByID:     req.TenantInfo.UserID,
		StartedAt:         s.now(),
		Version:           1,
	}
	if req.Owner.Actor != nil {
		if person := req.Owner.Actor.PersonUserID(); person.IsNotNil() {
			run.RequestedByID = person
		}
	}

	return run
}

func (s *Service) runAnalysis(
	ctx context.Context,
	attempt *analysisAttempt,
	req *agentextensionports.AnalysisRequest,
) (*agentextensionports.AnalysisOutcome, error) {
	call := attempt.call
	sandbox, err := call.provider.StartSandbox(ctx, &agentextensionports.SandboxRequest{
		Settings: call.values,
		Metadata: map[string]string{analysisMetadataRun: attempt.run.ID.String()},
	})
	if err != nil {
		return nil, s.failAnalysis(ctx, attempt, call.failing(err, "start a sandbox"))
	}
	attempt.sandbox = sandbox
	attempt.started = time.Now()
	defer s.stopSandbox(ctx, call, sandbox.ID)

	attempt.run.VendorSandboxID = sandbox.ID
	if updated, updateErr := s.analyses.Update(ctx, attempt.run); updateErr == nil {
		attempt.run = updated
	} else {
		s.l.Error("the vendor started a sandbox Trenova could not record",
			zap.String("analysisRun", attempt.run.ID.String()), zap.Error(updateErr))
	}

	files := make([]agentextensionports.SandboxFile, 0, len(req.Inputs))
	for idx := range req.Inputs {
		input := &req.Inputs[idx]
		files = append(files, agentextensionports.SandboxFile{
			Path:    agentextension.DatasetPath(input.Name, input.Extension),
			Content: input.Content,
		})
	}
	if err = call.provider.WriteFiles(ctx, sandbox, files); err != nil {
		return nil, s.failAnalysis(ctx, attempt, call.failing(err, "load the data into the sandbox"))
	}

	execution, err := call.provider.RunCode(ctx, &agentextensionports.RunCodeRequest{
		Sandbox: sandbox,
		Code:    req.Code,
		Timeout: time.Duration(call.settings.RunSeconds) * time.Second,
	})
	if err != nil {
		return nil, s.failAnalysis(ctx, attempt, call.failing(err, "run the code"))
	}

	return s.finishAnalysis(ctx, attempt, execution), nil
}

func (s *Service) stopSandbox(ctx context.Context, call *analysisCall, sandboxID string) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), analysisStopTimeout)
	defer cancel()

	if err := call.provider.StopSandbox(stopCtx, call.values, sandboxID); err != nil {
		s.l.Warn("could not stop an analysis sandbox; it ends at its own lifetime",
			zap.String("sandbox", sandboxID), zap.Error(err))
	}
}

// usageOf is how long the sandbox ran and what that cost, the start of every
// outcome. A sandbox that never started cost nothing.
func usageOf(attempt *analysisAttempt) *agentextension.AnalysisRunOutcome {
	if attempt.sandbox == nil || attempt.started.IsZero() {
		return &agentextension.AnalysisRunOutcome{}
	}
	ran := time.Since(attempt.started)

	return &agentextension.AnalysisRunOutcome{
		DurationMs: ran.Milliseconds(),
		CostUSD:    attempt.call.provider.CostOf(attempt.sandbox, ran),
	}
}

// failAnalysis records a run the sandbox could not carry out and returns the
// error the agent is told.
func (s *Service) failAnalysis(
	ctx context.Context,
	attempt *analysisAttempt,
	failure *providerFailure,
) error {
	err := s.providerError(failure)
	outcome := usageOf(attempt)
	outcome.Status = agentextension.AnalysisRunStatusFailed
	outcome.Failure = err.Error()
	s.recordAnalysis(ctx, attempt, outcome)

	return err
}

func (s *Service) finishAnalysis(
	ctx context.Context,
	attempt *analysisAttempt,
	execution *agentextensionports.SandboxExecution,
) *agentextensionports.AnalysisOutcome {
	outcome := usageOf(attempt)
	outcome.Stdout = execution.Stdout
	outcome.Stderr = execution.Stderr
	outcome.Error = execution.Error
	outcome.Results = textResults(execution.Results)

	switch {
	case execution.TimedOut:
		outcome.Status = agentextension.AnalysisRunStatusTimedOut
		outcome.Failure = "The code was still running after " +
			strconv.Itoa(attempt.call.settings.RunSeconds) + " seconds, so it was stopped."
	case execution.Incomplete:
		outcome.Status = agentextension.AnalysisRunStatusFailed
		outcome.Failure = "The code stopped without finishing, most often because it ran the " +
			"sandbox out of memory."
	case execution.Error != nil:
		outcome.Status = agentextension.AnalysisRunStatusErrored
	default:
		outcome.Status = agentextension.AnalysisRunStatusSucceeded
	}

	charts := s.keepCharts(ctx, attempt.run, execution.Results)
	outcome.ChartCount = len(charts)
	run := s.recordAnalysis(ctx, attempt, outcome)

	return &agentextensionports.AnalysisOutcome{
		RunID:      run.ID,
		Status:     run.Status,
		Datasets:   run.Datasets,
		Stdout:     run.Stdout,
		Stderr:     run.Stderr,
		Results:    run.Results,
		Charts:     charts,
		Error:      outcome.Error,
		Failure:    run.Failure,
		DurationMs: run.DurationMs,
	}
}

// recordAnalysis writes how a run ended and settles what it cost. A failure
// to write it is logged, not returned: the code has run either way.
func (s *Service) recordAnalysis(
	ctx context.Context,
	attempt *analysisAttempt,
	outcome *agentextension.AnalysisRunOutcome,
) *agentextension.AnalysisRun {
	run := attempt.run
	run.Finish(outcome, s.now())
	if updated, err := s.analyses.Update(ctx, run); err == nil {
		run = updated
	} else {
		s.l.Error("failed to record how an analysis ended",
			zap.String("analysisRun", run.ID.String()), zap.Error(err))
	}

	failed := run.Status == agentextension.AnalysisRunStatusFailed ||
		run.Status == agentextension.AnalysisRunStatusTimedOut
	s.settle(ctx, attempt.metered, settledCall{day: attempt.day, failed: failed, cost: run.CostUSD})

	return run
}

func textResults(results []agentextensionports.SandboxResult) []string {
	texts := make([]string, 0, len(results))
	for idx := range results {
		if len(results[idx].PNG) == 0 {
			if text := strings.TrimSpace(results[idx].Text); text != "" {
				texts = append(texts, text)
			}
		}
	}

	return texts
}

// keepCharts stores the images the code displayed, in the order it showed
// them, up to the most a run keeps. An image that is not a PNG or is too
// large is left out.
func (s *Service) keepCharts(
	ctx context.Context,
	run *agentextension.AnalysisRun,
	results []agentextensionports.SandboxResult,
) []agentextensionports.AnalysisChartSummary {
	charts := make([]*agentextension.AnalysisChart, 0, agentextension.MaxAnalysisCharts)
	for idx := range results {
		image := results[idx].PNG
		if len(charts) == agentextension.MaxAnalysisCharts {
			break
		}
		if !bytes.HasPrefix(image, pngSignature) || len(image) > agentextension.MaxAnalysisChartBytes {
			continue
		}
		charts = append(charts, &agentextension.AnalysisChart{
			RunID:          run.ID,
			Position:       len(charts) + 1,
			OrganizationID: run.OrganizationID,
			BusinessUnitID: run.BusinessUnitID,
			Title: stringutils.TruncateRunes(strings.TrimSpace(results[idx].Title),
				agentextension.MaxAnalysisChartTitle),
			Image: image,
		})
	}
	if len(charts) == 0 {
		return []agentextensionports.AnalysisChartSummary{}
	}

	if err := s.analyses.SaveCharts(ctx, charts); err != nil {
		s.l.Error("failed to keep an analysis's charts",
			zap.String("analysisRun", run.ID.String()), zap.Error(err))
		return []agentextensionports.AnalysisChartSummary{}
	}

	summaries := make([]agentextensionports.AnalysisChartSummary, 0, len(charts))
	for _, chart := range charts {
		summaries = append(summaries, agentextensionports.AnalysisChartSummary{
			Position: chart.Position,
			Title:    chart.Title,
		})
	}

	return summaries
}

func (s *Service) ListAnalysisRuns(
	ctx context.Context,
	req *agentextensionports.ListAnalysisRunsRequest,
) (*agentextensionports.AnalysisRunListResponse, error) {
	if req.Limit <= 0 || req.Limit > analysisRunListLimit {
		req.Limit = analysisRunListLimit
	}

	runs, err := s.analyses.List(ctx, req)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list analyses").WithInternal(err)
	}

	views := make([]agentextensionports.AnalysisRunView, 0, len(runs))
	for _, run := range runs {
		views = append(views, analysisRunView(run))
	}
	response := &agentextensionports.AnalysisRunListResponse{Items: views}
	if len(runs) == req.Limit {
		response.NextBefore = runs[len(runs)-1].StartedAt
	}

	return response, nil
}

func analysisRunView(run *agentextension.AnalysisRun) agentextensionports.AnalysisRunView {
	return agentextensionports.AnalysisRunView{
		ID:            run.ID,
		ExtensionType: run.ExtensionType,
		Status:        run.Status,
		Purpose:       run.Purpose,
		Code:          run.Code,
		Datasets:      run.Datasets,
		Stdout:        run.Stdout,
		Stderr:        run.Stderr,
		Results:       run.Results,
		ErrorName:     run.ErrorName,
		ErrorValue:    run.ErrorValue,
		Traceback:     run.Traceback,
		ChartCount:    run.ChartCount,
		Failure:       run.Failure,
		DurationMs:    run.DurationMs,
		CostUSD:       run.CostUSD,
		AgentRunID:    run.AgentRunID,
		ThreadID:      run.ThreadID,
		RequestedByID: run.RequestedByID,
		StartedAt:     run.StartedAt,
		FinishedAt:    run.FinishedAt,
	}
}

// AnalysisChart is one of a run's charts, for the person the analysis was run
// for or someone who oversees agents. Anyone else is told it does not exist.
func (s *Service) AnalysisChart(
	ctx context.Context,
	req *agentextensionports.AnalysisChartRequest,
) (*agentextension.AnalysisChart, error) {
	notFound := errortypes.NewNotFoundError("That chart was not found")

	run, err := s.analyses.Get(ctx, req.TenantInfo, req.RunID)
	if errortypes.IsNotFoundError(err) {
		return nil, notFound
	}
	if err != nil {
		return nil, err
	}
	if !req.Administers && run.RequestedByID != req.TenantInfo.UserID {
		return nil, notFound
	}
	if req.Position < 1 || req.Position > run.ChartCount {
		return nil, notFound
	}

	chart, err := s.analyses.GetChart(ctx, req.TenantInfo, run.ID, req.Position)
	if errortypes.IsNotFoundError(err) {
		return nil, notFound
	}
	if err != nil {
		return nil, err
	}

	return chart, nil
}
