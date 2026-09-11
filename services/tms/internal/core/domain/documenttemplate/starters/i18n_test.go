package starters_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderStarter renders one channel of a built-in starter in a given language, which is
// the exact path a worker takes when it sends a driver a notification.
func renderStarter(
	t *testing.T,
	kind documenttemplate.Kind,
	channel documenttemplate.Channel,
	locale i18n.Locale,
	data any,
) string {
	t.Helper()

	starter, err := starters.ForLocale(kind, locale)
	require.NoError(t, err)

	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)

	compiled, diags := engine.Parse(&templateengine.ParseRequest{
		Field:   channel.Field(),
		Kind:    string(kind),
		Channel: channel.Engine(),
		Source:  starter.Channel(channel),
		Locale:  locale,
	})
	require.False(t, diags.HasErrors(), "%v", diags)

	out, err := engine.Render(t.Context(), compiled, data)
	require.NoError(t, err)
	return out
}

func TestDriverNotificationRendersInTheDriversLanguage(t *testing.T) {
	t.Parallel()

	kind := documenttemplate.KindNotificationLoadAssigned

	assert.Equal(t, "New load assigned",
		renderStarter(t, kind, documenttemplate.ChannelNotificationTitle, i18n.EN, nil))
	assert.Equal(t, "Nueva carga asignada",
		renderStarter(t, kind, documenttemplate.ChannelNotificationTitle, i18n.ES, nil))
	assert.Equal(t, "已指派新貨載",
		renderStarter(t, kind, documenttemplate.ChannelNotificationTitle, i18n.ZhTW, nil))
	assert.Equal(t, "已分配新货载",
		renderStarter(t, kind, documenttemplate.ChannelNotificationTitle, i18n.ZhCN, nil))
}

func TestNotificationBodyInterpolatesInTheTargetLanguage(t *testing.T) {
	t.Parallel()

	data := map[string]any{"ShipmentProNumber": "P-10482", "StopCount": 3}

	spanish := renderStarter(t, documenttemplate.KindNotificationLoadAssigned,
		documenttemplate.ChannelNotificationBody, i18n.ES, data)

	assert.Contains(t, spanish, "Se le asignó la carga P-10482.")
	assert.Contains(t, spanish, "Tiene 3 paradas.")
	assert.Contains(t, spanish, "Abra Dash para ver las paradas y los detalles.")
	assert.NotContains(t, spanish, "You have been assigned")
}

func TestSingularAndPluralBothReadCorrectly(t *testing.T) {
	t.Parallel()

	one := renderStarter(t, documenttemplate.KindNotificationLoadAssigned,
		documenttemplate.ChannelNotificationBody, i18n.ES,
		map[string]any{"ShipmentProNumber": "P-1", "StopCount": 1})
	assert.Contains(t, one, "Tiene 1 parada.")

	// Chinese has one form, so the same count must not pick an English-style singular.
	chinese := renderStarter(t, documenttemplate.KindNotificationLoadAssigned,
		documenttemplate.ChannelNotificationBody, i18n.ZhTW,
		map[string]any{"ShipmentProNumber": "P-1", "StopCount": 1})
	assert.Contains(t, chinese, "共有 1 個停靠點。")
}

func TestEmailSubjectIsTranslated(t *testing.T) {
	t.Parallel()

	data := map[string]any{"CompanyName": "Trenova Freight"}

	assert.Equal(t, "Restablezca su contraseña de Trenova Freight",
		renderStarter(t, documenttemplate.KindPasswordResetEmail,
			documenttemplate.ChannelSubject, i18n.ES, data))
	assert.Equal(t, "重設您的 Trenova Freight 密碼",
		renderStarter(t, documenttemplate.KindPasswordResetEmail,
			documenttemplate.ChannelSubject, i18n.ZhTW, data))
}

func TestEveryStarterStillLoadsForEveryLocale(t *testing.T) {
	t.Parallel()

	registry := documenttemplate.NewRegistry()
	for _, locale := range i18n.Supported() {
		for _, def := range registry.All() {
			_, err := starters.ForLocale(def.Kind, locale)
			require.NoError(t, err, "starter %s missing for %s", def.Kind, locale)
		}
	}
}
