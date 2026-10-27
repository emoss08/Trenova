package board

import (
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
	numberDecimals   = 1
	percentDecimals  = 1
)

func FormatValue(value *Value) string {
	if value == nil {
		return ""
	}
	number := value.Number.InexactFloat64()
	switch value.Kind {
	case ValueCount:
		return floatutils.FormatGrouped(number, 0)
	case ValueMoney:
		return analysis.FormatValue(number, analysis.ValueMoney, value.Currency)
	case ValuePercent:
		return floatutils.FormatFractionAsPercent(number, percentDecimals)
	case ValueDuration:
		return FormatDuration(value.Number.IntPart())
	case ValueText:
		return value.Text
	default:
		return floatutils.FormatGrouped(number, numberDecimals)
	}
}

func FormatDuration(seconds int64) string {
	if seconds < 0 {
		seconds = -seconds
	}
	days := seconds / secondsPerDay
	hours := seconds % secondsPerDay / secondsPerHour
	minutes := seconds % secondsPerHour / secondsPerMinute
	switch {
	case days > 0 && hours > 0:
		return strconv.FormatInt(days, 10) + "d " + strconv.FormatInt(hours, 10) + "h"
	case days > 0:
		return strconv.FormatInt(days, 10) + "d"
	case hours > 0 && minutes > 0:
		return strconv.FormatInt(hours, 10) + "h " + strconv.FormatInt(minutes, 10) + "m"
	case hours > 0:
		return strconv.FormatInt(hours, 10) + "h"
	default:
		return strconv.FormatInt(minutes, 10) + "m"
	}
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

func RecordsPhrase(resource permission.Resource, count int) string {
	noun := resource.Noun()
	if count != 1 {
		noun = stringutils.PluralNoun(noun)
	}

	return strconv.Itoa(count) + " " + noun
}

func SourcePhrase(source *Source) string {
	switch source.Kind {
	case SourceRecords, SourceActivity:
		if source.Resource == "" {
			return ""
		}
		phrase := RecordsPhrase(source.Resource, source.Count)
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
	default:
		return ""
	}
}

func strconvQuote(title string) string {
	if title == "" {
		return ""
	}

	return "“" + title + "”"
}
