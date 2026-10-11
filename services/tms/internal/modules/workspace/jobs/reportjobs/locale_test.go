package reportjobs

import (
	"testing"

	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
)

func TestRunCompilesInTheRequestersLanguage(t *testing.T) {
	assert.Equal(
		t,
		i18n.ES,
		i18n.FromContext(withRunLocale(t.Context(), &PreparedRun{Locale: "es"})),
	)
	assert.Equal(
		t,
		i18n.ZhTW,
		i18n.FromContext(withRunLocale(t.Context(), &PreparedRun{Locale: "zh-TW"})),
	)
}

func TestRunWithoutALocaleKeepsTheDefault(t *testing.T) {
	assert.Equal(t, i18n.Default, i18n.FromContext(withRunLocale(t.Context(), &PreparedRun{})))
	assert.Equal(
		t,
		i18n.Default,
		i18n.FromContext(withRunLocale(t.Context(), &PreparedRun{Locale: "klingon"})),
	)
}
