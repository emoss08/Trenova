package formulatemplateresolver

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/formulamapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatereviewservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatestudioservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxDraftNameLength        = 100
	maxDraftDescriptionLength = 4000
)

func formulaDraft(
	input *gqlmodel.FormulaDraftInput,
) (formulastudioports.FormulaStudioDraft, error) {
	multiErr := errortypes.NewMultiError()
	draft := formulastudioports.FormulaStudioDraft{
		Name:        strings.TrimSpace(gqlscalar.StringValue(input.Name)),
		Description: strings.TrimSpace(gqlscalar.StringValue(input.Description)),
		Source:      input.Source,
	}

	if raw := strings.TrimSpace(gqlscalar.StringValue(input.TemplateID)); raw != "" {
		templateID, err := pulid.Parse(raw)
		if err != nil {
			multiErr.Add("templateId", errortypes.ErrInvalid, "Invalid formula template ID")
		}
		draft.TemplateID = templateID
	}
	switch {
	case strings.TrimSpace(input.Source) == "":
		multiErr.Add("source", errortypes.ErrRequired, "Source is required")
	case len(input.Source) > tarifftypes.MaxSourceLength:
		multiErr.Add("source", errortypes.ErrInvalid, "Source cannot exceed 64 KiB")
	}
	if utf8.RuneCountInString(draft.Name) > maxDraftNameLength {
		multiErr.Add("name", errortypes.ErrInvalid, "Name cannot be longer than 100 characters")
	}
	if utf8.RuneCountInString(draft.Description) > maxDraftDescriptionLength {
		multiErr.Add(
			"description",
			errortypes.ErrInvalid,
			"Description cannot be longer than 4,000 characters",
		)
	}
	if multiErr.HasErrors() {
		return draft, multiErr
	}
	return draft, nil
}

func explainInputValues(values map[string]any) (tarifftypes.Inputs, error) {
	if len(values) > formulatemplatestudioservice.MaxPreviewInputs {
		return nil, errortypes.NewValidationError(
			"inputs",
			errortypes.ErrInvalid,
			"Too many input values",
		)
	}
	return formulamapping.InputValues(values)
}

func explainTestName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" ||
		utf8.RuneCountInString(name) > formulatemplatestudioservice.MaxTestNameLength {
		return "", errortypes.NewValidationError(
			"testName",
			errortypes.ErrInvalid,
			"Name the test to explain, in at most 100 characters",
		)
	}
	return name, nil
}

func wireTrace(raw rules.RawJSON) rules.RawJSON {
	if raw.IsNone() {
		return rules.RawJSON("[]")
	}
	return raw
}

func previewToModel(
	result *formulatemplateports.PreviewResult,
) (*gqlmodel.FormulaPreviewResult, error) {
	if result == nil {
		return nil, nil //nolint:nilnil // a preview that did not run is a GraphQL null
	}
	diagnostics, err := formulamapping.WireList(result.Diagnostics)
	if err != nil {
		return nil, err
	}
	choices, err := formulamapping.WireList(result.Choices)
	if err != nil {
		return nil, err
	}
	tests, err := formulamapping.WireList(result.Tests)
	if err != nil {
		return nil, err
	}
	out := &gqlmodel.FormulaPreviewResult{
		Valid:       result.Valid,
		Diagnostics: diagnostics,
		Trace:       wireTrace(result.Trace),
		Lookups:     wireTrace(result.Lookups),
		Choices:     choices,
		Tests:       tests,
	}
	if result.Bill != nil {
		if out.Bill, err = formulamapping.Wire(result.Bill); err != nil {
			return nil, err
		}
	}
	if result.Refusal != nil {
		if out.Refusal, err = formulamapping.Wire(result.Refusal); err != nil {
			return nil, err
		}
	}
	if result.Error != nil {
		out.Error = &gqlmodel.FormulaPreviewError{
			Code:    result.Error.Code,
			Message: result.Error.Message,
			Range:   formulamapping.Range(result.Error.Range),
		}
	}
	return out, nil
}

func explainToModel(
	resp *formulastudioports.ExplainFormulaResponse,
) (*gqlmodel.FormulaExplainResult, error) {
	preview, err := previewToModel(resp.Preview)
	if err != nil {
		return nil, err
	}
	items := make([]*gqlmodel.FormulaTraceNote, 0, len(resp.Items))
	for i := range resp.Items {
		item := &resp.Items[i]
		items = append(items, &gqlmodel.FormulaTraceNote{
			Index: item.Index,
			Range: formulamapping.Range(item.Range),
			Says:  item.Says,
		})
	}
	return &gqlmodel.FormulaExplainResult{Preview: preview, Items: items}, nil
}

func suggestedTestsToModel(
	resp *formulastudioports.SuggestFormulaTestsResponse,
) (*gqlmodel.FormulaSuggestedTestsResult, error) {
	tests := make([]*gqlmodel.FormulaSuggestedTest, 0, len(resp.Tests))
	for i := range resp.Tests {
		test := &resp.Tests[i]
		failures, err := formulamapping.WireList(test.Failures)
		if err != nil {
			return nil, err
		}
		tests = append(tests, &gqlmodel.FormulaSuggestedTest{
			Name:     test.Name,
			Block:    test.Block,
			Status:   string(test.Status),
			Failures: failures,
		})
	}
	return &gqlmodel.FormulaSuggestedTestsResult{Tests: tests}, nil
}

func explainTestToModel(
	resp *formulastudioports.ExplainFormulaTestResponse,
) (*gqlmodel.FormulaExplainTestResult, error) {
	out := &gqlmodel.FormulaExplainTestResult{
		Verdict:   gqlmodel.FormulaTestVerdict(resp.Verdict),
		Why:       resp.Why,
		Generated: resp.Generated,
	}
	if resp.Test != nil {
		test, err := formulamapping.Wire(resp.Test)
		if err != nil {
			return nil, err
		}
		out.Test = test
	}
	return out, nil
}

func descriptionsToModel(descriptions map[string]string) map[string]any {
	out := make(map[string]any, len(descriptions))
	for name, doc := range descriptions {
		out[name] = doc
	}
	return out
}

func backtestSummaryToModel(
	summary *formulatemplateports.BacktestSummary,
) *gqlmodel.FormulaBacktestSummary {
	if summary == nil {
		return nil
	}
	return &gqlmodel.FormulaBacktestSummary{
		ShipmentCount:       summary.ShipmentCount,
		EvaluatedCount:      summary.EvaluatedCount,
		ChangedCount:        summary.ChangedCount,
		IncreasedCount:      summary.IncreasedCount,
		DecreasedCount:      summary.DecreasedCount,
		ErrorCount:          summary.ErrorCount,
		CurrentErrorCount:   summary.CurrentErrorCount,
		CandidateErrorCount: summary.CandidateErrorCount,
		RefusedCount:        summary.RefusedCount,
		AdjustedCount:       summary.AdjustedCount,
		CurrentTotal:        summary.CurrentTotal.String(),
		CandidateTotal:      summary.CandidateTotal.String(),
		TotalDelta:          summary.TotalDelta.String(),
		TotalDeltaPct:       summary.TotalDeltaPct.String(),
		MaxIncrease:         summary.MaxIncrease.String(),
		MaxDecrease:         summary.MaxDecrease.String(),
	}
}

func backtestToModel(
	resp *formulatemplatestudioservice.BacktestResponse,
) *gqlmodel.FormulaBacktestResult {
	results := make([]*gqlmodel.FormulaBacktestShipment, 0, len(resp.Results))
	for _, result := range resp.Results {
		if result == nil {
			continue
		}
		results = append(results, &gqlmodel.FormulaBacktestShipment{
			ShipmentID:       result.ShipmentID.String(),
			ProNumber:        result.ProNumber,
			CurrentSource:    gqlmodel.FormulaBacktestCurrentSource(result.CurrentSource),
			CurrentAmount:    result.CurrentAmount.String(),
			CandidateAmount:  result.CandidateAmount.String(),
			Delta:            result.Delta.String(),
			DeltaPct:         result.DeltaPct.String(),
			CurrentError:     gqlscalar.NonEmptyPtr(result.CurrentError),
			CandidateError:   gqlscalar.NonEmptyPtr(result.CandidateError),
			CandidateRefused: result.CandidateRefused,
			Adjusted:         result.Adjusted,
		})
	}
	return &gqlmodel.FormulaBacktestResult{
		Results:   results,
		Summary:   backtestSummaryToModel(&resp.Summary),
		Truncated: resp.Truncated,
	}
}

func studioContextToModel(
	studio *formulatemplatestudioservice.StudioContext,
) (*gqlmodel.FormulaStudioContext, error) {
	tariffCtx, err := formulamapping.Wire(studio.Context)
	if err != nil {
		return nil, err
	}
	libraries := make([]*gqlmodel.FormulaStudioLibrary, 0, len(studio.Libraries))
	for i := range studio.Libraries {
		library := &studio.Libraries[i]
		libraries = append(libraries, &gqlmodel.FormulaStudioLibrary{
			Path:          library.Path,
			LibraryID:     library.LibraryID.String(),
			Name:          library.Name,
			Description:   library.Description,
			VersionNumber: int(library.VersionNumber),
		})
	}
	return &gqlmodel.FormulaStudioContext{
		Context:     tariffCtx,
		Libraries:   libraries,
		TablesStamp: studio.TablesStamp,
	}, nil
}

func tableNeeds(inputs []*gqlmodel.FormulaTableNeedInput) []rules.TariffNeed {
	needs := make([]rules.TariffNeed, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		needs = append(needs, rules.TariffNeed{Table: input.Table, Keys: input.Keys})
	}
	return needs
}

func tableCellsToModel(cells *tariff.TableCells) (*gqlmodel.FormulaTableCells, error) {
	tables := make([]*gqlmodel.FormulaTableCellsTable, 0, len(cells.Tables))
	for i := range cells.Tables {
		table := &cells.Tables[i]
		wire, err := formulamapping.WireList(table.Cells)
		if err != nil {
			return nil, err
		}
		tables = append(tables, &gqlmodel.FormulaTableCellsTable{
			Code:   table.Code,
			Cells:  wire,
			Loaded: table.Loaded,
		})
	}
	return &gqlmodel.FormulaTableCells{Stamp: cells.Stamp, Tables: tables}, nil
}

func importConflict(conflict *gqlmodel.FormulaTemplateImportConflict) string {
	if conflict == nil {
		return ""
	}
	if *conflict == gqlmodel.FormulaTemplateImportConflictRename {
		return formulatemplateimportservice.ImportConflictRename
	}
	return formulatemplateimportservice.ImportConflictReject
}

func importPayloads(
	inputs []*gqlmodel.ImportFormulaTemplatePayloadInput,
) ([]*formulatemplateimportservice.ImportTemplatePayload, error) {
	payloads := make([]*formulatemplateimportservice.ImportTemplatePayload, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		sourceTemplateID, err := formulamapping.ParseOptionalID(
			input.SourceTemplateID,
			"sourceTemplateId",
			"Invalid formula template ID",
		)
		if err != nil {
			return nil, err
		}
		payload := &formulatemplateimportservice.ImportTemplatePayload{
			Name:                input.Name,
			Description:         gqlscalar.StringValue(input.Description),
			Source:              input.Source,
			SourceVersionNumber: gqlscalar.Int64Ptr(input.SourceVersionNumber),
		}
		if sourceTemplateID.IsNotNil() {
			payload.SourceTemplateID = &sourceTemplateID
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func renamedToModel(renamed map[string]string) map[string]any {
	if len(renamed) == 0 {
		return nil
	}
	out := make(map[string]any, len(renamed))
	for from, to := range renamed {
		out[from] = to
	}
	return out
}

func readinessToModel(
	resp *formulatemplatereviewservice.ReadinessResponse,
) *gqlmodel.FormulaReadiness {
	checks := make([]*gqlmodel.FormulaReadinessCheck, 0, len(resp.Checks))
	for i := range resp.Checks {
		check := &resp.Checks[i]
		checks = append(
			checks,
			formulamapping.ReadinessCheck(check.Key, check.Label, check.Status, check.Detail),
		)
	}
	return formulamapping.Readiness(&formulamapping.ReadinessParams{
		CanSubmit:        resp.CanSubmit,
		CanApprove:       resp.CanApprove,
		Checks:           checks,
		TestTotal:        resp.TestTotal,
		TestPassed:       resp.TestPassed,
		TestFailing:      resp.TestFailing,
		Coverage:         resp.Coverage,
		PendingSchedules: scheduledWindows(resp.PendingSchedules),
	})
}

func scheduledWindows(
	windows []formulatemplatereviewservice.ScheduledWindow,
) []*gqlmodel.FormulaScheduledWindow {
	out := make([]*gqlmodel.FormulaScheduledWindow, 0, len(windows))
	for _, window := range windows {
		out = append(
			out,
			formulamapping.ScheduledWindow(window.VersionNumber, window.EffectiveFrom),
		)
	}
	return out
}

func reviewDiffToModel(
	resp *formulatemplatereviewservice.ReviewDiffResponse,
) *gqlmodel.FormulaReviewDiff {
	return formulamapping.ReviewDiff(&formulamapping.ReviewDiffParams{
		HasApprovedBase:  resp.HasApprovedBase,
		BaseVersion:      resp.BaseVersion,
		CurrentVersion:   resp.CurrentVersion,
		BaseSource:       resp.BaseSource,
		CurrentSource:    resp.CurrentSource,
		BaseImports:      resp.BaseImports,
		CurrentImports:   resp.CurrentImports,
		Changes:          resp.Changes,
		ChangeCount:      resp.ChangeCount,
		PendingSchedules: scheduledWindows(resp.PendingSchedules),
	})
}

func reviewersToModel(
	reviewers []formulatemplatereviewservice.Reviewer,
) []*gqlmodel.FormulaReviewer {
	out := make([]*gqlmodel.FormulaReviewer, 0, len(reviewers))
	for i := range reviewers {
		reviewer := &reviewers[i]
		out = append(out, formulamapping.Reviewer(formulamapping.ReviewerParams{
			ID:            reviewer.ID,
			Name:          reviewer.Name,
			Username:      reviewer.Username,
			ProfilePicURL: reviewer.ProfilePicURL,
		}))
	}
	return out
}

func standardsToModel(
	standards []formulatemplateimportservice.StandardTemplate,
) []*formulatemplateimportservice.StandardTemplate {
	out := make([]*formulatemplateimportservice.StandardTemplate, 0, len(standards))
	for i := range standards {
		out = append(out, &standards[i])
	}
	return out
}
