package formulatemplateservice

import (
	"context"
	goErrors "errors"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
)

var versionDiffIgnoreFields = []string{
	"id",
	"createdAt",
	"createdById",
	"versionNumber",
	"changeMessage",
	"changeSummary",
}

func expressionErrorMessage(ctx context.Context, err error) string {
	var multiErr *errortypes.MultiError
	if goErrors.As(err, &multiErr) && len(multiErr.Errors) > 0 {
		return errortypes.LocalizedText(ctx, multiErr.Errors[0])
	}
	return errortypes.LocalizedText(ctx, err)
}

func clearApprovalFields(template *formulatemplate.FormulaTemplate) {
	template.SubmittedByID = nil
	template.SubmittedAt = nil
	template.ApprovedByID = nil
	template.ApprovedAt = nil
	template.ReviewComment = ""
}

// carryApprovalFields keeps the review stamps the workflow wrote, whatever an
// update payload says about them.
func carryApprovalFields(template, original *formulatemplate.FormulaTemplate) {
	template.SubmittedByID = original.SubmittedByID
	template.SubmittedAt = original.SubmittedAt
	template.ApprovedByID = original.ApprovedByID
	template.ApprovedAt = original.ApprovedAt
	template.ReviewComment = original.ReviewComment
}

func extractVersionPair(
	versions []*formulatemplate.FormulaTemplateVersion,
	fromNum, toNum int64,
) (fromVer, toVer *formulatemplate.FormulaTemplateVersion) {
	for _, v := range versions {
		switch v.VersionNumber {
		case fromNum:
			fromVer = v
		case toNum:
			toVer = v
		}
	}

	return fromVer, toVer
}

func buildLineage(
	template *formulatemplate.FormulaTemplate,
	forkedTemplates []*formulatemplate.FormulaTemplate,
) *formulatemplate.ForkLineage {
	lineage := &formulatemplate.ForkLineage{
		TemplateID:       template.ID,
		TemplateName:     template.Name,
		SourceTemplateID: template.SourceTemplateID,
		SourceVersion:    template.SourceVersionNumber,
		ForkedTemplates:  []formulatemplate.ForkLineage{},
	}

	for _, forked := range forkedTemplates {
		lineage.ForkedTemplates = append(lineage.ForkedTemplates, formulatemplate.ForkLineage{
			TemplateID:       forked.ID,
			TemplateName:     forked.Name,
			SourceTemplateID: forked.SourceTemplateID,
			SourceVersion:    forked.SourceVersionNumber,
		})
	}

	return lineage
}
