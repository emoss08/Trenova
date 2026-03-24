package formulatemplateservice

import "github.com/emoss08/trenova/internal/core/domain/formulatemplate"

var versionDiffIgnoreFields = []string{
	"id",
	"createdAt",
	"createdById",
	"versionNumber",
	"changeMessage",
	"changeSummary",
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
