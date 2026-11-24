package i18n

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLongDateWritesTheDayAsEachLanguageDoes(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.November, 3, 0, 0, 0, 0, time.UTC)
	for locale, want := range map[Locale]string{
		EN:          "Nov 3, 2026",
		ES:          "3 nov 2026",
		ZhTW:        "2026年11月3日",
		ZhCN:        "2026年11月3日",
		Locale("x"): "Nov 3, 2026",
	} {
		assert.Equal(t, want, LongDate(locale, day), "locale %s", locale)
	}
}

func TestDayLeavesTheYearOutWhenItIsThisYear(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)
	today := time.Date(2026, time.November, 24, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "Nov 2", Day(EN, day, today))
	assert.Equal(t, "2 nov", Day(ES, day, today))
	assert.Equal(t, "11月2日", Day(ZhTW, day, today))
	assert.Equal(t, "Nov 2, 2026", Day(EN, day, today.AddDate(1, 0, 0)))
}
