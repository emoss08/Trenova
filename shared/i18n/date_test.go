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
