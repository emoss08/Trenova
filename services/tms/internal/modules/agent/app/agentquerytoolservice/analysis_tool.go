package agentquerytoolservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	paramAnalysisCode     = "code"
	paramAnalysisPurpose  = "purpose"
	paramAnalysisDatasets = "datasets"
	paramDatasetName      = "name"
	paramDatasetRun       = "reportRunId"
	paramDatasetDocument  = "documentId"

	maxAnalysisOutputRunes    = 6000
	maxAnalysisTracebackRunes = 3000

	analysisChartNote = "The charts are shown to the person beside your reply; say what they " +
		"show and what it means, without describing how they were drawn."
	analysisDoneNote = "Figures come from the datasets as they were when each report ran. " +
		"Give the answer in plain words with the figures that matter; do not paste the code."
	analysisErrorNote = "The code raised an error. Fix it and run it again; do not run the " +
		"same code unchanged. Open a dataset at its path, and check a column's id in the " +
		"report's columns before using it."
	analysisStoppedNote = "Tell the person if it cannot be done; otherwise narrow the data or " +
		"simplify the code and run it again."
)

var (
	errAnalysisNeedsAPerson = errors.New(
		"analyze_data reads report runs and files a person gave you, and this run has no person",
	)
	errAnalysisReportRead = errors.New(
		"you do not have permission to read reports, so their results cannot be analyzed",
	)
)

type analyzeDataTool struct {
	reports     reportRunner
	files       documentports.ConversationFileReader
	analyzer    agentextensionports.DataAnalyzer
	permissions permissionports.PermissionEngine
}

func newAnalyzeDataTool(
	reports reportRunner,
	files documentports.ConversationFileReader,
	analyzer agentextensionports.DataAnalyzer,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &analyzeDataTool{
		reports:     reports,
		files:       files,
		analyzer:    analyzer,
		permissions: permissions,
	}
}

func (t *analyzeDataTool) Name() string { return agentextension.ToolAnalyzeData }

func (t *analyzeDataTool) Description() string {
	return "Run Python over report results and files the person attached, to work out what a " +
		"report alone does not show: margin by lane over a quarter, which customers drift past " +
		"terms, where detention clusters. pandas, numpy and matplotlib are available, and the " +
		"code cannot reach the internet. Each dataset is written to /home/user/data/<name>.csv " +
		"(or .xlsx, .json, .txt for a file) before the code runs; a report run's CSV has one " +
		"column per report column, headed by its id, with raw values: decimals as exact text, " +
		"times as unix seconds. Print what you need to read back, and show a chart with " +
		"plt.show() to put it beside your reply. Run the report with run_report first and use " +
		"its runId; each call starts afresh, so load what it needs every time."
}

func (t *analyzeDataTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramAnalysisPurpose: map[string]any{
				toolschema.KeyType: "string",
				toolschema.KeyDescription: "What the analysis answers, in one plain sentence the " +
					"person will read, such as \"Margin by lane for Q3\".",
				toolschema.KeyMaxLength: 500,
			},
			paramAnalysisCode: map[string]any{
				toolschema.KeyType: "string",
				toolschema.KeyDescription: "The Python to run, as one cell. Read each dataset " +
					"from its path, print the figures, and plt.show() any chart.",
				toolschema.KeyMaxLength: agentextension.MaxAnalysisCodeRunes,
			},
			paramAnalysisDatasets: map[string]any{
				toolschema.KeyType: "array",
				toolschema.KeyDescription: "The data the code reads, at most five. Give each a " +
					"name and exactly one of reportRunId or documentId.",
				toolschema.KeyMaxItems: agentextension.MaxAnalysisDatasets,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						paramDatasetName: map[string]any{
							toolschema.KeyType: "string",
							toolschema.KeyDescription: "The file's name in the sandbox: lower-case " +
								"letters, digits and underscores, starting with a letter, such as lanes.",
							toolschema.KeyMaxLength: agentextension.MaxAnalysisDatasetName,
						},
						paramDatasetRun: agenttoolschema.OfKinds(map[string]any{
							toolschema.KeyType: "string",
							toolschema.KeyDescription: "A finished report run you ran for this " +
								"person, from run_report. Its rows are written as CSV.",
						}, permission.KindReportRun),
						paramDatasetDocument: agenttoolschema.RecordIDText(
							permission.ResourceDocument,
							"A CSV, XLSX, JSON or text file the person attached to this "+
								"conversation, from read_attachment.",
						),
					},
					toolschema.KeyRequired:             []string{paramDatasetName},
					toolschema.KeyAdditionalProperties: false,
				},
			},
		},
		toolschema.KeyRequired:             []string{paramAnalysisPurpose, paramAnalysisCode},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *analyzeDataTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceDataAnalysis,
		reads:    agent.ExternalReadMarked,
		source:   agent.TaintSourceAttachment,
		rationale: "Runs code over report results and files the person already read, in a " +
			"sandbox that cannot reach the network; it changes nothing, and it carries the " +
			"taint of any attached file it reads.",
	})
}

func (t *analyzeDataTool) Prerequisites() []string {
	return []string{"run_report"}
}

func (t *analyzeDataTool) SearchTerms() []string {
	return []string{
		"analyze", "analysis", "chart", "graph", "plot", "trend", "python", "pandas",
		"spreadsheet", "calculate", "margin by lane", "compare periods", "statistics",
	}
}

type analysisDatasetAnswer struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
	Label  string `json:"label,omitempty"`
	Rows   int    `json:"rows,omitempty"`
}

type analysisErrorAnswer struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Traceback string `json:"traceback,omitempty"`
}

type analysisAnswer struct {
	AnalysisRunID string                                     `json:"analysisRunId"`
	Status        string                                     `json:"status"`
	Purpose       string                                     `json:"purpose"`
	Datasets      []analysisDatasetAnswer                    `json:"datasets"`
	Output        string                                     `json:"output,omitempty"`
	Results       []string                                   `json:"results,omitempty"`
	Charts        []agentextensionports.AnalysisChartSummary `json:"charts,omitempty"`
	Error         *analysisErrorAnswer                       `json:"error,omitempty"`
	Failure       string                                     `json:"failure,omitempty"`
	Note          string                                     `json:"note"`

	attachments []agent.RecordRef
}

// TaintedRecords names the attached files the analysis read: what the person
// attached may have been written outside the organization.
func (a *analysisAnswer) TaintedRecords() []agent.RecordRef { return a.attachments }

type datasetRequest struct {
	name       string
	reportRun  pulid.ID
	documentID pulid.ID
}

func datasetsFrom(raw any) ([]datasetRequest, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list", paramAnalysisDatasets)
	}
	if len(items) > agentextension.MaxAnalysisDatasets {
		return nil, fmt.Errorf("give at most %d datasets", agentextension.MaxAnalysisDatasets)
	}

	requests := make([]datasetRequest, 0, len(items))
	for idx, item := range items {
		fields, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("dataset %d must be an object with a name", idx+1)
		}
		name := strings.TrimSpace(optionalString(fields, paramDatasetName))
		if err := agentextension.CheckDatasetName(name); err != nil {
			return nil, err
		}
		run, hasRun, err := optionalPulid(fields, paramDatasetRun)
		if err != nil {
			return nil, err
		}
		document, hasDocument, err := optionalPulid(fields, paramDatasetDocument)
		if err != nil {
			return nil, err
		}
		if hasRun == hasDocument {
			return nil, fmt.Errorf(
				"dataset %s needs exactly one of reportRunId or documentId", name)
		}
		requests = append(requests, datasetRequest{name: name, reportRun: run, documentID: document})
	}

	return requests, nil
}

func (t *analyzeDataTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	purpose, err := requireString(params.Params, paramAnalysisPurpose)
	if err != nil {
		return nil, err
	}
	code, err := requireString(params.Params, paramAnalysisCode)
	if err != nil {
		return nil, err
	}
	if err = agentextension.CheckAnalysisCode(code); err != nil {
		return nil, err
	}
	requests, err := datasetsFrom(params.Params[paramAnalysisDatasets])
	if err != nil {
		return nil, err
	}

	inputs, attachments, err := t.resolve(ctx, params, requests)
	if err != nil {
		return nil, err
	}

	outcome, err := t.analyzer.Analyze(ctx, &agentextensionports.AnalysisRequest{
		TenantInfo: tenantOf(params),
		Purpose:    purpose,
		Code:       code,
		Inputs:     inputs,
		Owner: agentextensionports.AnalysisOwner{
			Actor:        params.Actor,
			DefinitionID: params.AgentDefinitionID,
			RunID:        params.RunID,
			ThreadID:     params.ThreadID,
		},
	})
	if err != nil {
		return nil, err
	}

	return answerOf(purpose, outcome, attachments), nil
}

// resolve reads each dataset as the caller: a report run only if this person
// ran it and may read reports, a file only if they attached it to this
// conversation. Nothing is read for anyone else.
func (t *analyzeDataTool) resolve(
	ctx context.Context,
	params *agentports.QueryToolParams,
	requests []datasetRequest,
) ([]agentextensionports.AnalysisInput, []agent.RecordRef, error) {
	inputs := make([]agentextensionports.AnalysisInput, 0, len(requests))
	attachments := make([]agent.RecordRef, 0)
	if len(requests) == 0 {
		return inputs, attachments, nil
	}

	person := personOf(params.Actor)
	if person.IsNil() {
		return nil, nil, errAnalysisNeedsAPerson
	}

	readsReports := false
	for idx := range requests {
		request := &requests[idx]
		var (
			input *agentextensionports.AnalysisInput
			err   error
		)
		if request.reportRun.IsNotNil() {
			if !readsReports {
				if err = t.authorizeReports(ctx, params.Actor); err != nil {
					return nil, nil, err
				}
				readsReports = true
			}
			input, err = t.reportDataset(ctx, params, person, request)
		} else {
			input, err = t.fileDataset(ctx, params, person, request)
			if err == nil {
				attachments = append(attachments, agent.RecordRef{
					EntityType: agent.TaintEntityDocument,
					ID:         request.documentID.String(),
				})
			}
		}
		if err != nil {
			return nil, nil, err
		}
		inputs = append(inputs, *input)
	}

	return inputs, attachments, nil
}

func (t *analyzeDataTool) authorizeReports(
	ctx context.Context,
	actor *permissionports.RequestActor,
) error {
	result, err := t.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceReport.String(),
		Operation:      permission.OpRead,
	})
	if err != nil {
		return fmt.Errorf("reading reports could not be authorized: %w", err)
	}
	if !result.Allowed {
		return errAnalysisReportRead
	}

	return nil
}

func (t *analyzeDataTool) reportDataset(
	ctx context.Context,
	params *agentports.QueryToolParams,
	person pulid.ID,
	request *datasetRequest,
) (*agentextensionports.AnalysisInput, error) {
	runRequest := &reporting.GetRunRequest{
		Request: reportingRequestFor(params),
		RunID:   request.reportRun,
	}
	notYours := fmt.Errorf(
		"report run %s is not one you ran for this person; run the report with run_report and use its runId",
		request.reportRun,
	)

	run, err := t.reports.GetRun(ctx, runRequest)
	if err != nil {
		if isNotFound(err) {
			return nil, notYours
		}
		return nil, err
	}
	if run.RequestedByID != person {
		return nil, notYours
	}

	envelope, err := t.reports.ReadRunRows(ctx, runRequest)
	if err != nil {
		return nil, err
	}
	var content bytes.Buffer
	if err = envelope.WriteCSV(&content); err != nil {
		return nil, fmt.Errorf("the rows of report run %s could not be written as CSV: %w",
			request.reportRun, err)
	}

	return &agentextensionports.AnalysisInput{
		Name:      request.name,
		Source:    agentextension.AnalysisDatasetReportRun,
		SourceID:  request.reportRun,
		Label:     envelope.Meta.Title,
		Extension: "csv",
		Content:   content.Bytes(),
		Rows:      len(envelope.Rows),
	}, nil
}

func (t *analyzeDataTool) fileDataset(
	ctx context.Context,
	params *agentports.QueryToolParams,
	person pulid.ID,
	request *datasetRequest,
) (*agentextensionports.AnalysisInput, error) {
	notAttached := fmt.Errorf(attachmentOutsideThread, request.documentID)
	if params.ThreadID.IsNil() {
		return nil, notAttached
	}

	file, err := t.files.ReadConversationFile(ctx, documentports.GetDocumentByIDRequest{
		ID:         request.documentID,
		TenantInfo: tenantOf(params),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, notAttached
		}
		return nil, err
	}
	if file.Document == nil || !file.Document.AttachedBy(params.ThreadID, person) {
		return nil, notAttached
	}

	extension := strings.TrimPrefix(strings.ToLower(path.Ext(file.Document.OriginalName)), ".")

	return &agentextensionports.AnalysisInput{
		Name:      request.name,
		Source:    agentextension.AnalysisDatasetAttachment,
		SourceID:  request.documentID,
		Label:     file.Document.OriginalName,
		Extension: extension,
		Content:   file.Data,
	}, nil
}

func answerOf(
	purpose string,
	outcome *agentextensionports.AnalysisOutcome,
	attachments []agent.RecordRef,
) *analysisAnswer {
	answer := &analysisAnswer{
		AnalysisRunID: outcome.RunID.String(),
		Status:        outcome.Status.String(),
		Purpose:       strings.TrimSpace(purpose),
		Datasets:      make([]analysisDatasetAnswer, 0, len(outcome.Datasets)),
		Output:        analysisOutput(outcome.Stdout, outcome.Stderr),
		Results:       outcome.Results,
		Charts:        outcome.Charts,
		Failure:       outcome.Failure,
		attachments:   attachments,
	}
	for idx := range outcome.Datasets {
		dataset := &outcome.Datasets[idx]
		answer.Datasets = append(answer.Datasets, analysisDatasetAnswer{
			Name:   dataset.Name,
			Path:   dataset.Path,
			Source: dataset.Source.String(),
			Label:  dataset.Label,
			Rows:   dataset.Rows,
		})
	}
	if outcome.Error != nil {
		answer.Error = &analysisErrorAnswer{
			Name:      outcome.Error.Name,
			Value:     outcome.Error.Value,
			Traceback: stringutils.TruncateRunes(outcome.Error.Traceback, maxAnalysisTracebackRunes),
		}
	}

	switch outcome.Status {
	case agentextension.AnalysisRunStatusSucceeded:
		answer.Note = analysisDoneNote
		if len(outcome.Charts) > 0 {
			answer.Note += " " + analysisChartNote
		}
	case agentextension.AnalysisRunStatusErrored:
		answer.Note = analysisErrorNote
	case agentextension.AnalysisRunStatusFailed,
		agentextension.AnalysisRunStatusTimedOut,
		agentextension.AnalysisRunStatusRunning:
		answer.Note = strings.TrimSpace(outcome.Failure + " " + analysisStoppedNote)
	}

	return answer
}

// analysisOutput is what the code printed, the end kept when it is long: a
// summary is usually printed last.
func analysisOutput(stdout, stderr string) string {
	output := strings.TrimSpace(stdout)
	if warnings := strings.TrimSpace(stderr); warnings != "" {
		if output != "" {
			output += "\n"
		}
		output += "[stderr]\n" + warnings
	}

	runes := []rune(output)
	if len(runes) <= maxAnalysisOutputRunes {
		return output
	}

	return "…" + string(runes[len(runes)-maxAnalysisOutputRunes+1:])
}
