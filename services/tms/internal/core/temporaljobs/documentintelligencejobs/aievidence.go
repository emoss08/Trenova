package documentintelligencejobs

import (
	"strconv"
	"strings"

	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	aiEvidenceContextBytes = 60
	aiEvidenceMaxRunes     = 200
)

var moneyNoise = strings.NewReplacer("$", "", ",", "", "%", "", " ", "", " ", "")

func withFieldEvidence(
	result *services.AIExtractResult,
	pages []services.AIDocumentPage,
) *services.AIExtractResult {
	if result == nil || len(pages) == 0 || len(result.Fields) == 0 {
		return result
	}

	fields := make(map[string]services.AIDocumentField, len(result.Fields))
	for key, field := range result.Fields {
		if strings.TrimSpace(field.EvidenceExcerpt) == "" {
			page, excerpt := locateEvidence(field.Value, field.PageNumber, pages)
			field.EvidenceExcerpt = excerpt
			if field.PageNumber <= 0 {
				field.PageNumber = page
			}
		}
		fields[key] = field
	}

	located := *result
	located.Fields = fields
	return &located
}

func locateEvidence(
	value string,
	preferredPage int,
	pages []services.AIDocumentPage,
) (int, string) {
	needles := evidenceNeedles(value)
	if len(needles) == 0 {
		return 0, ""
	}

	for _, needle := range needles {
		for _, page := range pagesPreferring(preferredPage, pages) {
			index := stringutils.IndexFold(page.Text, needle)
			if index < 0 {
				continue
			}
			excerpt := stringutils.ExcerptAround(
				page.Text,
				index,
				index+len(needle),
				aiEvidenceContextBytes,
			)
			return page.PageNumber, stringutils.TruncateRunes(excerpt, aiEvidenceMaxRunes)
		}
	}

	return 0, ""
}

func pagesPreferring(pageNumber int, pages []services.AIDocumentPage) []services.AIDocumentPage {
	if pageNumber <= 0 {
		return pages
	}

	ordered := make([]services.AIDocumentPage, 0, len(pages))
	for _, page := range pages {
		if page.PageNumber == pageNumber {
			ordered = append(ordered, page)
		}
	}
	for _, page := range pages {
		if page.PageNumber != pageNumber {
			ordered = append(ordered, page)
		}
	}

	return ordered
}

func evidenceNeedles(value string) []string {
	text := strings.TrimSpace(value)
	if text == "" {
		return nil
	}

	needles := []string{text}
	amount, err := decimal.NewFromString(moneyNoise.Replace(text))
	if err != nil {
		return needles
	}

	fixed := amount.StringFixed(2)
	negative := strings.HasPrefix(fixed, "-")
	wholeText, cents, _ := strings.Cut(strings.TrimPrefix(fixed, "-"), ".")
	whole, err := strconv.ParseInt(wholeText, 10, 64)
	if err != nil {
		return needles
	}
	grouped := intutils.FormatWithCommas(whole)
	if cents != "00" {
		grouped += "." + cents
	}
	if negative {
		grouped = "-" + grouped
	}
	if !strings.EqualFold(grouped, text) {
		needles = append(needles, grouped)
	}

	return needles
}
