package documentextractionjobs

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/shared/stringutils"
)

func buildContentPages(
	content *documentcontent.Content,
	pages []*PageExtractionResult,
) []*documentcontent.Page {
	items := make([]*documentcontent.Page, 0, len(pages))
	for _, page := range pages {
		items = append(items, &documentcontent.Page{
			DocumentContentID:    content.ID,
			DocumentID:           content.DocumentID,
			OrganizationID:       content.OrganizationID,
			BusinessUnitID:       content.BusinessUnitID,
			PageNumber:           page.PageNumber,
			SourceKind:           page.SourceKind,
			ExtractedText:        page.Text,
			Markdown:             page.Markdown,
			OCRConfidence:        page.OCRConfidence,
			PreprocessingApplied: page.PreprocessingApplied,
			Width:                page.Width,
			Height:               page.Height,
			Metadata:             defaultMetadata(page.Metadata),
		})
	}

	return items
}

func finalizeExtraction(pages []*PageExtractionResult, maxExtractedChars int) *ExtractionResult {
	textParts := make([]string, 0, len(pages))
	pageCount := len(pages)
	nativeCount := 0
	ocrCount := 0
	weightedConfidence := 0.0
	weightedPages := 0.0

	for _, page := range pages {
		if trimmed := strings.TrimSpace(page.Text); trimmed != "" {
			textParts = append(textParts, trimmed)
		}
		switch page.SourceKind {
		case documentcontent.SourceKindOCR:
			ocrCount++
			weightedConfidence += page.OCRConfidence
			weightedPages++
		case documentcontent.SourceKindNative:
			nativeCount++
			weightedConfidence += maxConfidence
			weightedPages++
		case documentcontent.SourceKindMixed:
			nativeCount++
			ocrCount++
			weightedConfidence += (page.OCRConfidence + maxConfidence) / 2
			weightedPages++
		}
	}

	sourceKind := documentcontent.SourceKindNative
	switch {
	case ocrCount > 0 && nativeCount > 0:
		sourceKind = documentcontent.SourceKindMixed
	case ocrCount > 0:
		sourceKind = documentcontent.SourceKindOCR
	}

	if weightedPages > 0 && len(pages) > 0 {
		for idx := range pages {
			if pages[idx].Metadata == nil {
				pages[idx].Metadata = map[string]any{}
			}
			pages[idx].Metadata["documentAverageConfidence"] = clampConfidence(
				weightedConfidence / weightedPages,
			)
		}
	}

	return &ExtractionResult{
		Text:       stringutils.TruncateAndTrim(strings.Join(textParts, "\n\n"), maxExtractedChars),
		PageCount:  pageCount,
		SourceKind: sourceKind,
		Pages:      pages,
	}
}

func defaultMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}
