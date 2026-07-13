package formulatemplateservice

import (
	"reflect"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
)

var versionDiffIgnoreFields = []string{
	"id",
	"createdAt",
	"createdById",
	"versionNumber",
	"changeMessage",
	"changeSummary",
}

func clearApprovalFields(template *formulatemplate.FormulaTemplate) {
	template.SubmittedByID = nil
	template.SubmittedAt = nil
	template.ApprovedByID = nil
	template.ApprovedAt = nil
	template.ReviewComment = ""
}

func sanitizeResolvedVariables(variables map[string]any) map[string]any {
	if len(variables) == 0 {
		return nil
	}

	sanitized := make(map[string]any, len(variables))
	for name, value := range variables {
		if value != nil && reflect.TypeOf(value).Kind() == reflect.Func {
			continue
		}
		sanitized[name] = value
	}

	return sanitized
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
