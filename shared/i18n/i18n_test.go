package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag   string
		want  Locale
		known bool
	}{
		{"en", EN, true},
		{"es", ES, true},
		{"zh-TW", ZhTW, true},
		{"zh-tw", ZhTW, true},
		{"zh_TW", ZhTW, true},
		{"zh-CN", ZhCN, true},
		{"zh-Hant", ZhTW, true},
		{"zh-Hans", ZhCN, true},
		{"zh-HK", ZhTW, true},
		{"es-MX", ES, true},
		{"en-GB", EN, true},
		{"zh", ZhCN, true},
		{"fr", Default, false},
		{"", Default, false},
		{"   ", Default, false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			t.Parallel()
			got, known := Parse(tt.tag)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.known, known)
		})
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   Locale
	}{
		{"empty falls back", "", EN},
		{"single tag", "es", ES},
		{"quality ordering prefers higher q", "en;q=0.4,zh-TW;q=0.9", ZhTW},
		{"equal quality keeps header order", "es,zh-CN", ES},
		{"implicit quality outranks explicit lower", "zh-CN,en;q=0.8", ZhCN},
		{"skips unsupported languages", "de,fr;q=0.9,es;q=0.5", ES},
		{"zero quality is a refusal, not a preference", "es;q=0,zh-CN;q=0.1", ZhCN},
		{"wildcard yields the default", "*", EN},
		{"regional variant narrows to base", "es-419,en;q=0.2", ES},
		{"malformed quality is treated as default weight", "zh-TW;q=bogus,en;q=0.9", ZhTW},
		{"all unsupported falls back", "de,fr,it", EN},
		{"whitespace is tolerated", "  zh-TW ; q=0.9 ,  en ; q=0.1 ", ZhTW},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ParseAcceptLanguage(tt.header))
		})
	}
}

func TestSelectPlural(t *testing.T) {
	t.Parallel()

	assert.Equal(t, pluralOne, selectPlural(EN, 1))
	assert.Equal(t, pluralOther, selectPlural(EN, 0))
	assert.Equal(t, pluralOther, selectPlural(EN, 2))
	assert.Equal(t, pluralOne, selectPlural(ES, 1))
	assert.Equal(t, pluralOther, selectPlural(ES, 5))

	assert.Equal(t, pluralOther, selectPlural(ZhCN, 1),
		"Chinese has a single plural form; 1 must not select the English one-form")
	assert.Equal(t, pluralOther, selectPlural(ZhTW, 0))
	assert.Equal(t, pluralOther, selectPlural(ZhTW, 3))
}

func TestFormatPositional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		args    []any
		want    string
	}{
		{"no args returns message", "Save", nil, "Save"},
		{"single placeholder", `Delete "{0}"?`, []any{"Load 42"}, `Delete "Load 42"?`},
		{"multiple placeholders", "{0} of {1}", []any{3, 9}, "3 of 9"},
		{"repeated placeholder", "{0} and {0}", []any{"x"}, "x and x"},
		{"out of range index is left intact", "{3}", []any{"a"}, "{3}"},
		{"unmatched brace is literal", "100% { done", []any{"a"}, "100% { done"},
		{"non-numeric index is left intact", "{name}", []any{"a"}, "{name}"},
		{"float renders without trailing zeros", "{0} mi", []any{12.5}, "12.5 mi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, format(EN, tt.message, tt.args))
		})
	}
}

func TestFormatPlural(t *testing.T) {
	t.Parallel()

	message := "{0, plural, one {# shipment} other {# shipments}}"

	assert.Equal(t, "1 shipment", format(EN, message, []any{1}))
	assert.Equal(t, "0 shipments", format(EN, message, []any{0}))
	assert.Equal(t, "7 shipments", format(EN, message, []any{7}))

	spanish := "{0, plural, one {# envío} other {# envíos}}"
	assert.Equal(t, "1 envío", format(ES, spanish, []any{1}))
	assert.Equal(t, "4 envíos", format(ES, spanish, []any{4}))

	chinese := "{0, plural, other {# 個運單}}"
	assert.Equal(t, "1 個運單", format(ZhTW, chinese, []any{1}),
		"a Chinese catalog entry supplies only the other form")
}

func TestFormatPluralFallsBackToOther(t *testing.T) {
	t.Parallel()

	message := "{0, plural, other {# items}}"
	assert.Equal(t, "1 items", format(EN, message, []any{1}),
		"a missing one-form must fall back to other rather than dropping the text")
}

func TestFormatPluralWithSurroundingText(t *testing.T) {
	t.Parallel()

	message := "You have {0, plural, one {# message} other {# messages}} waiting"
	assert.Equal(t, "You have 1 message waiting", format(EN, message, []any{1}))
	assert.Equal(t, "You have 3 messages waiting", format(EN, message, []any{3}))
}

func TestTranslateFallsBackToSource(t *testing.T) {
	t.Parallel()

	const untranslated = "A message no catalog will ever carry"

	assert.Equal(t, untranslated, Translate(ES, untranslated))
	assert.Equal(t, untranslated, Translate(EN, untranslated))
	assert.Equal(t, "", Translate(ES, ""))
}

func TestTranslateFormatsFallback(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `Discard "Route 9" without saving?`,
		Translate(ZhTW, `Discard "{0}" without saving?`, "Route 9"),
		"an untranslated message must still interpolate its arguments")
}

func TestTranslateInvalidLocaleUsesDefault(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Save", Translate(Locale("klingon"), "Save"))
}

func TestFromContextDefaultsWhenAbsent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Default, FromContext(t.Context()))
}

func TestWithLocaleRoundTrips(t *testing.T) {
	t.Parallel()

	ctx := WithLocale(t.Context(), ZhTW)
	assert.Equal(t, ZhTW, FromContext(ctx))
	assert.Equal(t, "Save", T(ctx, "Save"))
}

func TestWithLocaleRejectsUnsupported(t *testing.T) {
	t.Parallel()

	ctx := WithLocale(t.Context(), Locale("fr"))
	assert.Equal(t, Default, FromContext(ctx),
		"an unsupported locale must not be stored; it would miss every catalog lookup")
}

func TestCatalogsLoad(t *testing.T) {
	t.Parallel()

	require.NoError(t, CatalogError())
	counts := Loaded()
	for _, locale := range Supported() {
		_, ok := counts[locale]
		assert.True(t, ok, "catalog missing for %s", locale)
	}
}
