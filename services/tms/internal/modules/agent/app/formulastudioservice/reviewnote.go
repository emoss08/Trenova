package formulastudioservice

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	maxRenderedChanges  = 30
	maxChangeValueRunes = 300
	reviewNoteMaxTokens = 1500
)

const reviewNoteSystem = "You are the formula assistant in Trenova's Formula Studio. Draft a " +
	"note from the author to the person who will review this formula template: at most 120 " +
	"words of plain prose, first what changed since the last approved version and why it " +
	"matters, then what re-rating recent shipments showed, when a backtest is given. Write " +
	"identifiers in backticks. Write it as the author, addressed to the reviewer, without a " +
	"greeting or a sign-off. " + amountsRule + " " + untrustedNote

func (s *Service) DraftReviewNote(
	ctx context.Context,
	req *formulastudioports.DraftFormulaReviewNoteRequest,
	actor *permissionports.RequestActor,
) (*formulastudioports.DraftFormulaReviewNoteResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	template, err := s.engine.GetByID(ctx, formulatemplateports.GetFormulaTemplateByIDRequest{
		TemplateID: req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	diff, err := s.engine.ReviewDiff(ctx, &formulatemplateservice.ReviewDiffRequest{
		TenantInfo: req.TenantInfo,
		TemplateID: req.TemplateID,
	})
	if err != nil {
		return nil, err
	}

	resp := &formulastudioports.DraftFormulaReviewNoteResponse{}
	if diff.HasApprovedBase && req.IncludeBacktest {
		resp.Backtest = s.backtestSaved(ctx, req, template)
	}

	var reply reviewNoteReply
	err = s.complete(ctx, &completion{
		tenant:     req.TenantInfo,
		definition: definition,
		feature:    aiusage.FeatureFormulaExplain,
		system:     reviewNoteSystem,
		sections: []llmports.ContextSection{
			{Title: "Template", Content: reviewTemplateText(template)},
			{Title: "Changes since the last approved version", Content: diffText(diff)},
			{Title: "Backtest of recent shipments", Content: backtestText(resp.Backtest)},
		},
		schema:     reviewNoteSchema(),
		schemaName: "formula_review_note",
		maxTokens:  reviewNoteMaxTokens,
	}, &reply)
	if err != nil {
		return nil, modelFailure(definition, err)
	}

	note := strings.TrimSpace(reply.Note)
	if note == "" {
		return nil, modelFailure(definition, llmports.ErrModelSchemaValidation)
	}

	resp.Note = stringutils.Ellipsize(note, formulatemplate.MaxReviewerNoteLength-1)
	resp.Generated = true
	return resp, nil
}

func (s *Service) backtestSaved(
	ctx context.Context,
	req *formulastudioports.DraftFormulaReviewNoteRequest,
	template *formulatemplate.FormulaTemplate,
) *formulatemplateports.BacktestSummary {
	result, err := s.engine.Backtest(ctx, &formulatemplateservice.BacktestRequest{
		TenantInfo: req.TenantInfo,
		TemplateID: req.TemplateID,
		Source:     template.Source,
	})
	if err != nil {
		s.l.Warn("the review note was drafted without a backtest", zap.Error(err))
		return nil
	}

	summary := result.Summary
	return &summary
}

type reviewNoteReply struct {
	Note string `json:"note"`
}

func reviewNoteSchema() map[string]any {
	return objectSchema(map[string]any{
		"note": typedField("string", "The note, at most 120 words"),
	}, "note")
}

func reviewTemplateText(template *formulatemplate.FormulaTemplate) string {
	return fmt.Sprintf(
		"Name: %s\nDescription: %s\nTests: %d\nVersion being submitted: v%d",
		template.Name,
		stringutils.WithDefault(strings.TrimSpace(template.Description), "(none)"),
		template.TestCount,
		template.CurrentVersionNumber,
	)
}

func diffText(diff *formulatemplateservice.ReviewDiffResponse) string {
	if !diff.HasApprovedBase {
		return "This is the template's first version; nothing has been approved before, so " +
			"all of it is new.\nSource:\n" + diff.CurrentSource
	}
	if diff.ChangeCount == 0 {
		return fmt.Sprintf("Nothing changed since v%d was approved.", diff.BaseVersion)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(
		"From v%d (approved) to v%d:\n",
		diff.BaseVersion,
		diff.CurrentVersion,
	))
	names := slices.Sorted(maps.Keys(diff.Changes))
	if len(names) > maxRenderedChanges {
		names = names[:maxRenderedChanges]
	}
	for _, name := range names {
		change := diff.Changes[name]
		builder.WriteString(fmt.Sprintf(
			"- %s: %s -> %s\n",
			name,
			changeValue(change.From),
			changeValue(change.To),
		))
	}
	if diff.BaseSource != diff.CurrentSource {
		builder.WriteString("Approved source:\n")
		builder.WriteString(diff.BaseSource)
		builder.WriteString("\nSubmitted source:\n")
		builder.WriteString(diff.CurrentSource)
	}

	return builder.String()
}

func changeValue(value any) string {
	if value == nil {
		return "empty"
	}
	if text, ok := value.(string); ok {
		return stringutils.OneLine(text, maxChangeValueRunes)
	}

	encoded, err := sonic.MarshalString(value)
	if err != nil {
		return stringutils.OneLine(fmt.Sprint(value), maxChangeValueRunes)
	}

	return stringutils.OneLine(encoded, maxChangeValueRunes)
}

func backtestText(summary *formulatemplateports.BacktestSummary) string {
	if summary == nil {
		return "No backtest: either nothing has been approved before or recent shipments " +
			"could not be re-rated."
	}

	return fmt.Sprintf(
		"Shipments re-rated: %d\nChanged: %d (%d up, %d down)\nFailed to rate: %d\nRefused: %d\n"+
			"Revenue on the approved version: %s\nRevenue on the submitted version: %s\n"+
			"Difference: %s (%s%%)\nLargest increase: %s\nLargest decrease: %s",
		summary.EvaluatedCount,
		summary.ChangedCount,
		summary.IncreasedCount,
		summary.DecreasedCount,
		summary.ErrorCount,
		summary.RefusedCount,
		summary.CurrentTotal.String(),
		summary.CandidateTotal.String(),
		summary.TotalDelta.String(),
		summary.TotalDeltaPct.String(),
		summary.MaxIncrease.String(),
		summary.MaxDecrease.String(),
	)
}
