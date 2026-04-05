package documentintelligencejobs

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/shared/intutils"
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

func thresholdImage(img image.Image, threshold uint8) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			lumaVal := ((299 * r) + (587 * g) + (114 * b) + 500) / 1000 >> 8
			luma := intutils.SafeUint32ToUint8(lumaVal)
			alpha := intutils.SafeUint32ToUint8(a >> 8)
			if luma >= threshold {
				dst.SetNRGBA(x, y, color.NRGBA{
					R: 255, G: 255, B: 255, A: alpha,
				})
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: 0, G: 0, B: 0, A: alpha,
			})
		}
	}

	return dst
}

//nolint:gocognit,funlen // TSV parsing with line-grouping state machine
func parseTesseractTSV(
	output string,
) (parsed string, avgConfidence float64, parseErr error) { //nolint:unparam // error return reserved for future format validation
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	if len(lines) <= 1 {
		return "", 0, nil
	}

	type lineKey struct {
		page  int
		block int
		par   int
		line  int
	}

	lineTexts := make([]string, 0)
	currentKey := lineKey{}
	currentWords := make([]string, 0, 8)
	totalConfidence := 0.0
	confidenceCount := 0.0
	seenHeader := false

	flush := func() {
		if len(currentWords) == 0 {
			return
		}
		lineTexts = append(lineTexts, strings.Join(currentWords, " "))
		currentWords = currentWords[:0]
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		cols := strings.Split(raw, "\t")
		if len(cols) < 12 {
			continue
		}
		if !seenHeader {
			seenHeader = true
			if strings.EqualFold(cols[0], "level") {
				continue
			}
		}

		level, err := strconv.Atoi(cols[0])
		if err != nil || level != 5 {
			continue
		}

		key := lineKey{}
		if key.page, err = strconv.Atoi(cols[1]); err != nil {
			continue
		}
		if key.block, err = strconv.Atoi(cols[2]); err != nil {
			continue
		}
		if key.par, err = strconv.Atoi(cols[3]); err != nil {
			continue
		}
		if key.line, err = strconv.Atoi(cols[4]); err != nil {
			continue
		}

		if currentKey != (lineKey{}) && key != currentKey {
			flush()
		}
		currentKey = key

		text := strings.TrimSpace(cols[11])
		if text == "" {
			continue
		}
		currentWords = append(currentWords, text)

		conf, err := strconv.ParseFloat(cols[10], 64)
		if err == nil && conf >= 0 {
			totalConfidence += conf / 100
			confidenceCount++
		}
	}

	flush()

	parsed = strings.TrimSpace(strings.Join(lineTexts, "\n"))
	if confidenceCount > 0 {
		avgConfidence = clampConfidence(totalConfidence / confidenceCount)
	}

	return parsed, avgConfidence, nil
}
