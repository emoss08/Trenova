package agentdefinition

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithinBusinessHours(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	tests := []struct {
		name     string
		on       bool
		zone     string
		fallback string
		at       time.Time
		want     bool
	}{
		{"off is always within", false, "", "", time.Date(2026, 10, 3, 3, 0, 0, 0, time.UTC), true},
		{"opening minute is within", true, "America/Chicago", "",
			time.Date(2026, 10, 3, 7, 0, 0, 0, chicago), true},
		{"closing minute is outside", true, "America/Chicago", "",
			time.Date(2026, 10, 3, 18, 0, 0, 0, chicago), false},
		{"night is outside", true, "America/Chicago", "",
			time.Date(2026, 10, 3, 23, 30, 0, 0, chicago), false},
		{"read in the organization's zone when the agent names none", true, "", "America/Chicago",
			// 13:00 UTC is 8 AM in Chicago, inside; it would be outside in UTC
			// only before 7.
			time.Date(2026, 10, 3, 13, 0, 0, 0, time.UTC), true},
		{"the agent's zone wins over the organization's", true, "Asia/Tokyo", "America/Chicago",
			time.Date(2026, 10, 3, 13, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &Definition{BusinessHoursOnly: tt.on, BusinessHoursTimezone: tt.zone}
			d.applyCapabilityDefaults()
			assert.Equal(t, tt.want, d.WithinBusinessHours(tt.at, tt.fallback))
		})
	}
}

func TestTurnToolOffAndOn(t *testing.T) {
	t.Parallel()

	d := &Definition{
		ToolNames:       []string{"post_invoices", "assign_biller"},
		ToolTiers:       map[string]agent.AutonomyTier{"post_invoices": agent.TierActWithApproval},
		ToolDailyLimits: map[string]int{"post_invoices": 5},
	}
	tiers := d.ToolTiers

	d.TurnToolOff("post_invoices")
	assert.Equal(t, []string{"assign_biller"}, d.ToolNames)
	assert.Equal(t, []string{"post_invoices"}, d.DisabledToolNames)
	assert.NotContains(t, d.ToolTiers, "post_invoices")
	assert.NotContains(t, d.ToolDailyLimits, "post_invoices")
	assert.Contains(t, tiers, "post_invoices", "a copy sharing the old map is left alone")

	d.TurnToolOn("post_invoices")
	assert.Equal(t, []string{"assign_biller", "post_invoices"}, d.ToolNames)
	assert.Empty(t, d.DisabledToolNames)
}

func TestForgetReheldToolsAndPruneTopics(t *testing.T) {
	t.Parallel()

	kept := pulid.MustNew(IDPrefix)
	dropped := pulid.MustNew(IDPrefix)
	d := &Definition{
		ToolNames:         []string{"void_invoice"},
		DisabledToolNames: []string{"void_invoice", "send_email"},
		DelegateIDs:       []pulid.ID{kept},
		DelegateTopics: map[string]string{
			kept.String():    " Pay rates ",
			dropped.String(): "Dispatch",
		},
	}

	d.ForgetReheldTools()
	d.PruneDelegateTopics()

	assert.Equal(t, []string{"send_email"}, d.DisabledToolNames)
	assert.Equal(t, map[string]string{kept.String(): "Pay rates"}, d.DelegateTopics)
	assert.Equal(t, "Pay rates", d.DelegateTopic(kept))
	assert.Empty(t, d.DelegateTopic(dropped))
}

func TestValidateCapabilities(t *testing.T) {
	t.Parallel()

	delegate := pulid.MustNew(IDPrefix)
	tests := []struct {
		name  string
		edit  func(d *Definition)
		field string
	}{
		{"zero items", func(d *Definition) { d.MaxChangeItems = 0 }, "maxChangeItems"},
		{"too many items", func(d *Definition) { d.MaxChangeItems = 10001 }, "maxChangeItems"},
		{"ends before it starts", func(d *Definition) {
			d.BusinessHoursStart, d.BusinessHoursEnd = 600, 540
		}, "businessHoursEnd"},
		{"unknown zone", func(d *Definition) { d.BusinessHoursTimezone = "Mars/Olympus" },
			"businessHoursTimezone"},
		{"topic for a stranger", func(d *Definition) {
			d.DelegateTopics = map[string]string{pulid.MustNew(IDPrefix).String(): "x"}
		}, "delegateTopics."},
		{"on and off at once", func(d *Definition) {
			d.DisabledToolNames = []string{"post_invoices"}
		}, "disabledToolNames[0]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &Definition{
				ToolNames:      []string{"post_invoices"},
				DelegateIDs:    []pulid.ID{delegate},
				DelegateTopics: map[string]string{delegate.String(): "Pay rates"},
			}
			d.applyCapabilityDefaults()
			tt.edit(d)
			multiErr := errortypes.NewMultiError()
			d.validateCapabilities(multiErr)
			require.True(t, multiErr.HasErrors())
			assert.Contains(t, multiErr.Error(), tt.field)
		})
	}

	ok := &Definition{
		DelegateIDs:    []pulid.ID{delegate},
		DelegateTopics: map[string]string{delegate.String(): "Pay rates"},
	}
	ok.applyCapabilityDefaults()
	multiErr := errortypes.NewMultiError()
	ok.validateCapabilities(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestFormatMinute(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "7 AM", FormatMinute(420))
	assert.Equal(t, "6 PM", FormatMinute(1080))
	assert.Equal(t, "12 AM", FormatMinute(0))
	assert.Equal(t, "12:30 PM", FormatMinute(750))
}
