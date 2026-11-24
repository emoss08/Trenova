package board

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardwording"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/inflect"
)

const (
	numberDecimals  = 1
	percentDecimals = 1
	centDecimals    = 2
	subCentDecimals = 3
)

func FormatValue(value *boardcatalog.Value) string {
	if value == nil {
		return ""
	}
	number := value.Number.InexactFloat64()
	switch value.Kind {
	case boardcatalog.ValueCount:
		return floatutils.FormatGrouped(number, 0)
	case boardcatalog.ValueMoney:
		return formatMoney(value, number)
	case boardcatalog.ValuePercent:
		return floatutils.FormatFractionAsPercent(number, percentDecimals)
	case boardcatalog.ValueDuration:
		return boardwording.FormatDuration(value.Number.IntPart())
	case boardcatalog.ValueDays:
		return FormatDays(value.Number.IntPart())
	case boardcatalog.ValueHours:
		return boardwording.FormatHours(value.Number.IntPart())
	case boardcatalog.ValueText:
		return value.Text
	case boardcatalog.ValueNumber:
	}

	return floatutils.FormatGrouped(number, numberDecimals)
}

func formatMoney(value *boardcatalog.Value, number float64) string {
	text := analysis.FormatValue(number, analysis.ValueMoney, value.Currency)
	if number == 0 || math.Abs(number) >= 1 || value.Number.Exponent() >= -subCentDecimals+1 {
		return text
	}
	cents := value.Number.Abs().StringFixed(subCentDecimals)
	cents = strings.TrimSuffix(cents, "0")

	return strings.Replace(text, value.Number.Abs().StringFixed(centDecimals), cents, 1)
}

func FormatDays(days int64) string {
	if days < 0 {
		days = -days
	}
	if days == 1 {
		return "1 day"
	}

	return strconv.FormatInt(days, 10) + " days"
}

func FormatMoment(at int64, location *time.Location) string {
	if at <= 0 {
		return ""
	}
	if location == nil {
		location = time.UTC
	}

	return time.Unix(at, 0).In(location).Format("Jan 2, 3:04 PM")
}

func FormatDay(at int64, location *time.Location) string {
	if at <= 0 {
		return ""
	}
	if location == nil {
		location = time.UTC
	}

	return time.Unix(at, 0).In(location).Format("Jan 2, 2006")
}

func FormatWords(words *boardreading.Words, location *time.Location) string {
	if words == nil {
		return ""
	}
	text := words.Text
	if strings.Contains(text, wordsSlot) {
		filled := ""
		switch {
		case words.Value != nil:
			filled = FormatValue(words.Value)
		case words.At > 0 && words.Day:
			filled = FormatDay(words.At, location)
		case words.At > 0:
			filled = FormatMoment(words.At, location)
		}
		text = strings.ReplaceAll(text, wordsSlot, filled)
	}
	if words.Place != "" {
		text += " " + words.Place
	}

	return strings.TrimSpace(text)
}

const wordsSlot = "{0}"

func RecordsPhrase(kind recordkind.Kind, count int) string {
	noun := kind.Noun()
	if count != 1 {
		noun = inflect.PluralNoun(noun)
	}

	return strconv.Itoa(count) + " " + noun
}

func SourcePhrase(source *Source) string {
	switch source.Kind {
	case SourceRecords, SourceActivity:
		if source.RecordKind == "" {
			return ""
		}
		phrase := RecordsPhrase(source.RecordKind, source.Count)
		if source.Kind == SourceActivity {
			return "the history of " + strings.TrimPrefix(phrase, "1 ")
		}
		return phrase
	case SourceAnalysis:
		return "the analysis " + strconvQuote(source.Title)
	case SourceTable:
		return "the table " + strconvQuote(source.Title)
	case SourceMetrics:
		return "Trenova's figures"
	case SourceChecks:
		return "the checks above"
	case SourceMixed:
		return "several sources"
	case SourceNone:
	}

	return ""
}

func strconvQuote(title string) string {
	if title == "" {
		return ""
	}

	return "“" + title + "”"
}
