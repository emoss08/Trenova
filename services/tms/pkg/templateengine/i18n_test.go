package templateengine

import (
	"testing"

	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderWith(t *testing.T, locale i18n.Locale, channel Channel, source string, data any) string {
	t.Helper()

	engine, err := New(Config{})
	require.NoError(t, err)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:   "body",
		Kind:    "test",
		Channel: channel,
		Source:  source,
		Locale:  locale,
	})
	require.False(t, diags.HasErrors(), "%v", diags)
	require.NotNil(t, compiled)

	out, err := engine.Render(t.Context(), compiled, data)
	require.NoError(t, err)
	return out
}

func TestTemplateTranslatesText(t *testing.T) {
	t.Parallel()

	source := `{{ t "New load assigned" }}`

	assert.Equal(t, "New load assigned", renderWith(t, i18n.EN, ChannelNotificationBody, source, nil))
	assert.Equal(t, "Nueva carga asignada", renderWith(t, i18n.ES, ChannelNotificationBody, source, nil))
	assert.Equal(t, "已指派新貨載", renderWith(t, i18n.ZhTW, ChannelNotificationBody, source, nil))
	assert.Equal(t, "已分配新货载", renderWith(t, i18n.ZhCN, ChannelNotificationBody, source, nil))
}

func TestTemplateTranslatesWithArguments(t *testing.T) {
	t.Parallel()

	source := `{{ t "You have been assigned load {0}." .Pro }}`
	data := map[string]any{"Pro": "P-10482"}

	assert.Equal(t, "Se le asignó la carga P-10482.", renderWith(t, i18n.ES, ChannelNotificationBody, source, data))
	assert.Equal(t, "已指派貨載 P-10482 給您。", renderWith(t, i18n.ZhTW, ChannelNotificationBody, source, data))
}

func TestTemplatePluralFollowsTheLocale(t *testing.T) {
	t.Parallel()

	source := `{{ t "It has {0, plural, one {# stop} other {# stops}}." .Stops }}`

	assert.Equal(t, "It has 1 stop.", renderWith(t, i18n.EN, ChannelNotificationBody, source, map[string]any{"Stops": 1}))
	assert.Equal(t, "It has 4 stops.", renderWith(t, i18n.EN, ChannelNotificationBody, source, map[string]any{"Stops": 4}))
	assert.Equal(t, "Tiene 1 parada.", renderWith(t, i18n.ES, ChannelNotificationBody, source, map[string]any{"Stops": 1}))
	assert.Equal(t, "Tiene 4 paradas.", renderWith(t, i18n.ES, ChannelNotificationBody, source, map[string]any{"Stops": 4}))

	// Chinese has one plural form, so a count of 1 must not pick the English one-form.
	assert.Equal(t, "共有 1 個停靠點。", renderWith(t, i18n.ZhTW, ChannelNotificationBody, source, map[string]any{"Stops": 1}))
	assert.Equal(t, "共有 4 個停靠點。", renderWith(t, i18n.ZhTW, ChannelNotificationBody, source, map[string]any{"Stops": 4}))
}

func TestTemplateUntranslatedFallsBackToSource(t *testing.T) {
	t.Parallel()

	source := `{{ t "A phrase nobody has translated yet" }}`
	assert.Equal(t, "A phrase nobody has translated yet",
		renderWith(t, i18n.ZhCN, ChannelNotificationBody, source, nil),
		"a missing translation must render English, never an empty body")
}

func TestTranslatedTextIsEscapedInHTML(t *testing.T) {
	t.Parallel()

	// t returns a plain string, so the contextual escaper still owns the output.
	// A translation carrying markup must not be able to inject it.
	out := renderWith(t, i18n.EN, ChannelEmailHTML, `<p>{{ t "{0}" .Name }}</p>`,
		map[string]any{"Name": "<script>alert(1)</script>"})

	assert.NotContains(t, out, "<script>")
	assert.Contains(t, out, "&lt;script&gt;")
}

func TestTIsAnAllowedFunction(t *testing.T) {
	t.Parallel()

	assert.Contains(t, AllowedFunctionNames(), "t",
		"the analyzer rejects calls outside the allowed set, so t must be listed")
}
