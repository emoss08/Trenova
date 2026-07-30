package starters

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file is the gate on the built-ins.
//
// A starter is what an organization that has customized nothing actually receives,
// so a malformed one is a broken invoice for every such organization. Every starter
// is parsed and rendered here under publish-strength rules, which means a bad
// built-in fails `go test` rather than reaching a customer.

func newEngine(t *testing.T) *templateengine.Engine {
	t.Helper()
	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)
	return engine
}

// TestEveryStarterPublishesCleanly is the load-bearing assertion: each starter must
// pass the same validation an administrator's own template has to pass, including
// the required-variable and sample-render checks.
func TestEveryStarterPublishesCleanly(t *testing.T) {
	registry := documenttemplate.NewRegistry()
	engine := newEngine(t)

	starters, err := All()
	require.NoError(t, err)
	require.Len(t, starters, len(documenttemplate.AllKinds()))

	for _, starter := range starters {
		def, ok := registry.Get(starter.Kind)
		require.True(t, ok)

		for _, channel := range def.Channels {
			t.Run(string(starter.Kind)+"/"+string(channel), func(t *testing.T) {
				source := starter.Channel(channel)
				require.NotEmpty(t, source,
					"%s has no content for channel %s", starter.Kind, channel)

				compiled, diags := engine.Parse(&templateengine.ParseRequest{
					Field:        channel.Field(),
					Kind:         string(starter.Kind),
					Channel:      channel.Engine(),
					Source:       source,
					AllowedPaths: registry.Paths(starter.Kind),
					Strict:       true,
				})
				require.False(t, diags.HasErrors(),
					"%s/%s does not validate: %s", starter.Kind, channel, diags.Summary())
				require.NotNil(t, compiled)

				sample, sampleErr := registry.SampleContext(starter.Kind)
				require.NoError(t, sampleErr)

				verifyDiags := engine.Verify(t.Context(), templateengine.VerifyRequest{
					Field:    channel.Field(),
					Channel:  channel.Engine(),
					Compiled: compiled,
					Sample:   sample,
				})
				assert.False(t, verifyDiags.HasErrors(),
					"%s/%s fails against sample data: %s",
					starter.Kind, channel, verifyDiags.Summary())
			})
		}
	}
}

// TestStartersSurviveHostileData is the reason the probe context exists. A starter
// that puts a text field somewhere its value gets neutralized would render a
// placeholder the first time a customer name contained an apostrophe.
func TestStartersSurviveHostileData(t *testing.T) {
	registry := documenttemplate.NewRegistry()
	engine := newEngine(t)

	starters, err := All()
	require.NoError(t, err)

	for _, starter := range starters {
		def, ok := registry.Get(starter.Kind)
		require.True(t, ok)

		probe, probeErr := registry.HostileProbeContext(starter.Kind)
		require.NoError(t, probeErr)

		for _, channel := range def.Channels {
			if !channel.Engine().IsHTML() {
				continue
			}

			t.Run(string(starter.Kind)+"/"+string(channel), func(t *testing.T) {
				compiled, diags := engine.Parse(&templateengine.ParseRequest{
					Field:        channel.Field(),
					Kind:         string(starter.Kind),
					Channel:      channel.Engine(),
					Source:       starter.Channel(channel),
					AllowedPaths: registry.Paths(starter.Kind),
				})
				require.False(t, diags.HasErrors(), diags.Summary())

				verifyDiags := engine.Verify(t.Context(), templateengine.VerifyRequest{
					Field:    channel.Field(),
					Channel:  channel.Engine(),
					Compiled: compiled,
					Sample:   probe,
				})
				assert.False(t, verifyDiags.HasErrors(),
					"%s/%s breaks on punctuation-heavy data, so it will break on a real "+
						"customer name: %s", starter.Kind, channel, verifyDiags.Summary())
			})
		}
	}
}

// TestRenderedStartersAreSanitary re-checks output rather than source. A starter can
// be clean on its own and still interpolate hostile data into a position that
// matters, which is what the render-time pass exists to catch.
func TestRenderedStartersAreSanitary(t *testing.T) {
	registry := documenttemplate.NewRegistry()
	engine := newEngine(t)

	starters, err := All()
	require.NoError(t, err)

	for _, starter := range starters {
		def, ok := registry.Get(starter.Kind)
		require.True(t, ok)

		probe, probeErr := registry.HostileProbeContext(starter.Kind)
		require.NoError(t, probeErr)

		for _, channel := range def.Channels {
			if !channel.Engine().IsHTML() {
				continue
			}

			t.Run(string(starter.Kind)+"/"+string(channel), func(t *testing.T) {
				compiled, diags := engine.Parse(&templateengine.ParseRequest{
					Field:        channel.Field(),
					Kind:         string(starter.Kind),
					Channel:      channel.Engine(),
					Source:       starter.Channel(channel),
					AllowedPaths: registry.Paths(starter.Kind),
				})
				require.False(t, diags.HasErrors(), diags.Summary())

				out, renderErr := engine.Render(t.Context(), compiled, probe)
				require.NoError(t, renderErr)

				opts := templateengine.DocumentSanitizeOptions()
				if channel == documenttemplate.ChannelEmailHTML {
					opts = templateengine.EmailSanitizeOptions()
				}
				outputDiags := templateengine.SanitizeHTML(channel.Field(), out, opts)
				assert.False(t, outputDiags.HasErrors(),
					"%s/%s renders unsafe output from hostile data: %s",
					starter.Kind, channel, outputDiags.Summary())

				// The probe value contains a script-ish fragment; if it survived
				// unescaped the escaper was bypassed.
				assert.NotContains(t, out, "</style><a:1",
					"probe data was not escaped")
			})
		}
	}
}

// TestEmailStartersUseInlineStyles guards the channel rule. A <style> block in an
// email is stripped by a large share of clients, so a starter relying on one ships a
// message that looks broken in those inboxes.
func TestEmailStartersUseInlineStyles(t *testing.T) {
	registry := documenttemplate.NewRegistry()

	starters, err := All()
	require.NoError(t, err)

	for _, starter := range starters {
		def, ok := registry.Get(starter.Kind)
		require.True(t, ok)
		if !def.HasChannel(documenttemplate.ChannelEmailHTML) {
			continue
		}

		t.Run(string(starter.Kind), func(t *testing.T) {
			assert.NotContains(t, strings.ToLower(starter.BodyHTML), "<style",
				"an email body must use inline style attributes")
			assert.Empty(t, starter.CSS,
				"a message kind has no stylesheet field; styles must be inline")
		})
	}
}

func TestDocumentStartersHaveAStylesheet(t *testing.T) {
	registry := documenttemplate.NewRegistry()

	starters, err := All()
	require.NoError(t, err)

	for _, starter := range starters {
		def, ok := registry.Get(starter.Kind)
		require.True(t, ok)
		if !def.HasChannel(documenttemplate.ChannelPDF) {
			continue
		}

		t.Run(string(starter.Kind), func(t *testing.T) {
			assert.NotEmpty(t, starter.CSS)

			cssDiags := templateengine.SanitizeCSS("cssContent", starter.CSS)
			assert.False(t, cssDiags.HasErrors(),
				"%s stylesheet is not acceptable: %s", starter.Kind, cssDiags.Summary())
		})
	}
}

// TestDocumentStartersAvoidPageBreakHazards checks the pagination properties the
// fixed-coordinate renderer could not express. Without them a charge row or an
// address block can be split across a page boundary.
func TestDocumentStartersAvoidPageBreakHazards(t *testing.T) {
	registry := documenttemplate.NewRegistry()

	starters, err := All()
	require.NoError(t, err)

	for _, starter := range starters {
		def, ok := registry.Get(starter.Kind)
		require.True(t, ok)
		if !def.HasChannel(documenttemplate.ChannelPDF) {
			continue
		}

		t.Run(string(starter.Kind), func(t *testing.T) {
			assert.Contains(t, starter.CSS, "break-inside: avoid",
				"a multi-page document must keep rows and blocks intact across breaks")
		})
	}
}

// TestDocumentStartersUseRealTableMarkup matters for a reason CSS alone cannot
// supply: a browser repeats <thead> on every page a table spills onto, and styled
// divs get no such treatment.
func TestDocumentStartersUseRealTableMarkup(t *testing.T) {
	registry := documenttemplate.NewRegistry()

	starters, err := All()
	require.NoError(t, err)

	for _, starter := range starters {
		def, ok := registry.Get(starter.Kind)
		require.True(t, ok)
		if !def.HasChannel(documenttemplate.ChannelPDF) {
			continue
		}
		// Only kinds that actually tabulate need this.
		if !strings.Contains(starter.BodyHTML, "<table") {
			continue
		}

		t.Run(string(starter.Kind), func(t *testing.T) {
			assert.Contains(t, starter.BodyHTML, "<tbody")
		})
	}
}

func TestPageSetup(t *testing.T) {
	t.Run("invoice is portrait letter", func(t *testing.T) {
		starter, err := For(documenttemplate.KindInvoicePDF)
		require.NoError(t, err)
		assert.Equal(t, documenttemplate.PageSizeLetter, starter.PageSize)
		assert.Equal(t, documenttemplate.OrientationPortrait, starter.Orientation)
	})

	t.Run("report is landscape with the historical margins", func(t *testing.T) {
		starter, err := For(documenttemplate.KindReportPDF)
		require.NoError(t, err)
		assert.Equal(t, documenttemplate.OrientationLandscape, starter.Orientation,
			"a report squeezed into portrait loses its rightmost columns")
		assert.InDelta(t, reportMarginMillimeters, starter.Margins.Top, 0.01)
	})

	t.Run("page setup is always valid", func(t *testing.T) {
		starters, err := All()
		require.NoError(t, err)
		for _, starter := range starters {
			assert.True(t, starter.PageSize.IsValid(), string(starter.Kind))
			assert.True(t, starter.Orientation.IsValid(), string(starter.Kind))
			assert.True(t, starter.Margins.IsValid(), string(starter.Kind))
		}
	})
}

func TestSubjectsAreSingleLine(t *testing.T) {
	starters, err := All()
	require.NoError(t, err)

	for _, starter := range starters {
		if starter.Subject == "" {
			continue
		}
		t.Run(string(starter.Kind), func(t *testing.T) {
			// A newline here would travel into a mail header.
			assert.NotContains(t, starter.Subject, "\n")
			assert.NotContains(t, starter.Subject, "\r")
			assert.Equal(t, strings.TrimSpace(starter.Subject), starter.Subject)
		})
	}
}

// TestNoOrphanedAssets keeps the directory honest in both directions: a stale file
// is dead weight, and a missing one breaks a kind.
func TestNoOrphanedAssets(t *testing.T) {
	registry := documenttemplate.NewRegistry()

	names, err := AssetNames()
	require.NoError(t, err)
	require.NotEmpty(t, names)

	expected := make([]string, 0, len(names))
	for _, def := range registry.All() {
		stem := def.StarterKey()
		if def.HasChannel(documenttemplate.ChannelPDF) ||
			def.HasChannel(documenttemplate.ChannelEmailHTML) {
			expected = append(expected, stem+extHTML)
		}
		if def.HasChannel(documenttemplate.ChannelPDF) {
			expected = append(expected, stem+extCSS)
		}
		if def.HasChannel(documenttemplate.ChannelEmailText) ||
			def.HasChannel(documenttemplate.ChannelNotificationBody) {
			expected = append(expected, stem+extText)
		}
		if def.HasChannel(documenttemplate.ChannelSubject) ||
			def.HasChannel(documenttemplate.ChannelNotificationTitle) {
			expected = append(expected, stem+extSubject)
		}
	}

	slices.Sort(expected)
	assert.Equal(t, expected, names,
		"the assets directory and the registry disagree")
}

func TestContentHashIsStableAndDistinct(t *testing.T) {
	first, err := Hash(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)
	second, err := Hash(documenttemplate.KindInvoicePDF)
	require.NoError(t, err)
	other, err := Hash(documenttemplate.KindReportPDF)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.NotEqual(t, first, other)
	assert.Len(t, first, 64)
}

func TestForUnknownKind(t *testing.T) {
	_, err := For(documenttemplate.Kind("nope.nope"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown template kind")
}

// TestStartersReferenceEveryRequiredPath is the property that makes the built-ins a
// usable starting point: an administrator who duplicates one and edits it must not
// be blocked at publish by a required field the built-in never had.
func TestStartersReferenceEveryRequiredPath(t *testing.T) {
	registry := documenttemplate.NewRegistry()
	engine := newEngine(t)

	starters, err := All()
	require.NoError(t, err)

	for _, starter := range starters {
		def, ok := registry.Get(starter.Kind)
		require.True(t, ok)

		required := registry.RequiredPaths(starter.Kind)
		if len(required) == 0 {
			continue
		}

		t.Run(string(starter.Kind), func(t *testing.T) {
			referenced := make(map[string]struct{}, 32)

			for _, channel := range def.Channels {
				compiled, diags := engine.Parse(&templateengine.ParseRequest{
					Field:        channel.Field(),
					Kind:         string(starter.Kind),
					Channel:      channel.Engine(),
					Source:       starter.Channel(channel),
					AllowedPaths: registry.Paths(starter.Kind),
				})
				require.False(t, diags.HasErrors(), diags.Summary())
				for _, path := range compiled.Paths {
					referenced[path] = struct{}{}
				}
			}

			for _, want := range required {
				_, direct := referenced[want]
				if direct {
					continue
				}
				child := false
				for path := range referenced {
					if strings.HasPrefix(path, want+".") {
						child = true
						break
					}
				}
				assert.True(t, child,
					"%s never references required path %q", starter.Kind, want)
			}
		})
	}
}

// notificationOutcomes renders both branches of a decision notification.
//
// A notification whose wording does not change with the decision is the failure
// that matters here: a driver told "time off approved" when it was denied acts on
// the wrong answer, and the branch is one {{ if }} away from being dropped by an
// administrator editing the template.
func TestDecisionNotificationsSayWhichWayItWent(t *testing.T) {
	engine := newEngine(t)
	registry := documenttemplate.NewRegistry()

	cases := []struct {
		kind     documenttemplate.Kind
		approved string
		denied   string
	}{
		{documenttemplate.KindNotificationPTOReviewed, "approved", "not approved"},
		{documenttemplate.KindNotificationExpenseReviewed, "approved", "rejected"},
		{documenttemplate.KindNotificationDisputeResolved, "in your favor", "stands"},
	}

	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			starter, err := For(tc.kind)
			require.NoError(t, err)

			render := func(approved bool) (title, body string) {
				sample, sampleErr := registry.SampleContext(tc.kind)
				require.NoError(t, sampleErr)
				data := withApproved(t, sample, approved)

				return renderChannel(t, engine, registry, starter,
						documenttemplate.ChannelNotificationTitle, data),
					renderChannel(t, engine, registry, starter,
						documenttemplate.ChannelNotificationBody, data)
			}

			approvedTitle, approvedBody := render(true)
			deniedTitle, deniedBody := render(false)

			assert.NotEqual(t, approvedTitle, deniedTitle,
				"the title must say which way the decision went")
			assert.Contains(t, approvedBody, tc.approved)
			assert.Contains(t, deniedBody, tc.denied)
		})
	}
}

// withApproved flips the Approved field on a copy of a sample context.
func withApproved(t *testing.T, sample any, approved bool) any {
	t.Helper()

	value := reflect.ValueOf(sample)
	clone := reflect.New(value.Type()).Elem()
	clone.Set(value)

	field := clone.FieldByName("Approved")
	require.True(t, field.IsValid(), "the family context has no Approved field")
	field.SetBool(approved)

	return clone.Interface()
}

func renderChannel(
	t *testing.T,
	engine *templateengine.Engine,
	registry *documenttemplate.Registry,
	starter *Starter,
	channel documenttemplate.Channel,
	data any,
) string {
	t.Helper()

	compiled, diags := engine.Parse(&templateengine.ParseRequest{
		Field:        channel.Field(),
		Kind:         string(starter.Kind),
		Channel:      channel.Engine(),
		Source:       starter.Channel(channel),
		AllowedPaths: registry.Paths(starter.Kind),
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	out, err := engine.Render(t.Context(), compiled, data)
	require.NoError(t, err)

	return out
}

// Every notification kind must be keyed by the event type that reaches the hub.
// A kind whose key does not match its event type renders nothing, and the
// notification is dropped rather than sent unstyled.
func TestNotificationKindsAreKeyedByEventType(t *testing.T) {
	registry := documenttemplate.NewRegistry()

	emitted := []string{
		"shipment_comment_mention",
		"shipment_comment_reply",
		"report_run_completed",
		"report_run_failed",
		"report_run_canceled",
		"report_run_delivered",
		"report_delivery_email_failed",
		"report_schedule_skipped",
		"dash.load_assigned",
		"dash.load_unassigned",
		"dash.pto_reviewed",
		"dash.credential_expiring",
		"dash.hos_alert",
		"dash.settlement_posted",
		"dash.settlement_paid",
		"dash.pay_held",
		"dash.expense_reviewed",
		"dash.dispute_resolved",
	}

	for _, eventType := range emitted {
		kind := documenttemplate.Kind("notification." + eventType)
		_, ok := registry.Get(kind)
		assert.True(t, ok,
			"%s is emitted by a service but no template kind is registered for it",
			eventType)

		starter, err := For(kind)
		require.NoError(t, err)
		assert.NotEmpty(t, starter.Subject, "%s has no title", kind)
		assert.NotEmpty(t, starter.BodyText, "%s has no message", kind)
		assert.Empty(t, starter.BodyHTML,
			"a notification renders as text; markup would show up as characters")
	}
}

// One event carries two outcomes: a schedule skipped this once, and a schedule
// disabled after failing repeatedly. An owner who reads "skipped" when the
// schedule is off will wait for a report that is never coming.
func TestScheduleSkippedSaysWhenTheScheduleWasDisabled(t *testing.T) {
	engine := newEngine(t)
	registry := documenttemplate.NewRegistry()

	starter, err := For(documenttemplate.KindNotificationReportScheduleSkipped)
	require.NoError(t, err)

	render := func(disabled bool) (title, body string) {
		data := documenttemplate.ReportNotificationContext{
			ReportName:          "Revenue by Customer",
			Reason:              "The report timed out.",
			Disabled:            disabled,
			ConsecutiveFailures: 3,
		}

		return renderChannel(t, engine, registry, starter,
				documenttemplate.ChannelNotificationTitle, data),
			renderChannel(t, engine, registry, starter,
				documenttemplate.ChannelNotificationBody, data)
	}

	skippedTitle, skippedBody := render(false)
	disabledTitle, disabledBody := render(true)

	assert.Contains(t, skippedTitle, "skipped")
	assert.Contains(t, disabledTitle, "disabled")
	assert.NotContains(t, skippedBody, "disabled")
	assert.Contains(t, disabledBody, "3 times in a row")
	assert.Contains(t, skippedBody, "The report timed out.",
		"the cause is the sentence the owner acts on")
}
