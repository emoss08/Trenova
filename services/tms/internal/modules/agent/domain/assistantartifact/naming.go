package assistantartifact

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
)

const (
	MaxTitleRunes = 120
	untitledTitle = "Untitled"
	titleQuotes   = "\"'“”‘’`"
	titleTrailers = ".:;,"
)

var titleMarkup = strings.NewReplacer(
	"[", "",
	"]", "",
	"*", "",
	"`", "",
	"\r", " ",
	"\n", " ",
	"\t", " ",
)

func NormalizeTitle(title string) string {
	cleaned := strings.Join(strings.Fields(titleMarkup.Replace(title)), " ")
	cleaned = strings.TrimLeft(cleaned, "#> "+titleQuotes)
	cleaned = strings.TrimRight(cleaned, titleQuotes+titleTrailers+" ")
	if cleaned == "" {
		return untitledTitle
	}

	return stringutils.TruncateRunes(textcase.CapitalizeFirst(cleaned), MaxTitleRunes)
}

func RecordNoun(entity string) string {
	return recordkind.Kind(entity).Noun()
}

func RecordTitle(entity, label string) string {
	return NormalizeTitle(RecordNoun(entity) + " " + strings.TrimSpace(label))
}

func RecordsTitle(entity string) string {
	return NormalizeTitle(inflect.PluralNoun(RecordNoun(inflect.SingularNoun(entity))))
}

func PhraseTitle(identifier string) string {
	return NormalizeTitle(textcase.HumanizeSnakeCasePhrase(identifier))
}
