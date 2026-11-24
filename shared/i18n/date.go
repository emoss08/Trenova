package i18n

import (
	"strconv"
	"time"
)

var shortMonths = map[Locale][12]string{
	EN: {"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
	ES: {"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sept", "oct", "nov", "dic"},
}

func LongDate(locale Locale, day time.Time) string {
	year := strconv.Itoa(day.Year())
	date := strconv.Itoa(day.Day())
	switch locale {
	case ZhTW, ZhCN:
		return year + "年" + strconv.Itoa(int(day.Month())) + "月" + date + "日"
	case ES:
		return date + " " + shortMonths[ES][day.Month()-1] + " " + year
	case EN:
		return shortMonths[EN][day.Month()-1] + " " + date + ", " + year
	default:
		return shortMonths[EN][day.Month()-1] + " " + date + ", " + year
	}
}

func ShortDate(locale Locale, day time.Time) string {
	date := strconv.Itoa(day.Day())
	switch locale {
	case ZhTW, ZhCN:
		return strconv.Itoa(int(day.Month())) + "月" + date + "日"
	case ES:
		return date + " " + shortMonths[ES][day.Month()-1]
	case EN:
		return shortMonths[EN][day.Month()-1] + " " + date
	default:
		return shortMonths[EN][day.Month()-1] + " " + date
	}
}

func Day(locale Locale, day, today time.Time) string {
	if day.Year() == today.Year() {
		return ShortDate(locale, day)
	}

	return LongDate(locale, day)
}
