package ifta_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
)

// An enum's Label() is English, and the English string is the catalog key, so an enum
// label localizes through the same lookup as everything else — no parallel table, and no
// way for a label to exist in one place and its translation in another.
func TestEnumLabelsLocalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		locale  i18n.Locale
		source  string
		mileage string
	}{
		{i18n.EN, "Manual entry", "Telematics"},
		{i18n.ES, "Captura manual", "Telemática"},
		{i18n.ZhTW, "手動輸入", "車聯網"},
		{i18n.ZhCN, "手动录入", "车联网"},
	}

	for _, tt := range tests {
		t.Run(string(tt.locale), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.source,
				i18n.Translate(tt.locale, ifta.MileageSourceManual.Label()))
			assert.Equal(t, tt.mileage,
				i18n.Translate(tt.locale, ifta.MileageSourceTelematics.Label()))
		})
	}
}

func TestProblemCodeLabelsLocalize(t *testing.T) {
	t.Parallel()

	label := ifta.ProblemMissingRate.Label()
	assert.Equal(t, "Missing tax rate", label)
	assert.Equal(t, "Falta la tasa impositiva", i18n.Translate(i18n.ES, label))
	assert.Equal(t, "缺少稅率", i18n.Translate(i18n.ZhTW, label))
	assert.Equal(t, "缺少税率", i18n.Translate(i18n.ZhCN, label))
}

func TestAnUntranslatedLabelStaysEnglish(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Q1 2026",
		i18n.Translate(i18n.ES, ifta.Period{Quarter: 1, Year: 2026}.Label()),
		"a composed label with no catalog entry must render as it always did")
}
