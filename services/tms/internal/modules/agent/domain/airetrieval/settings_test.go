package airetrieval

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func validSettings() *Settings {
	return DefaultSettings(pulid.MustNew("org_"), pulid.MustNew("bu_"))
}

func fieldsWithErrors(settings *Settings) []string {
	multiErr := errortypes.NewMultiError()
	settings.Validate(multiErr)

	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestSettingsValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Settings)
		fields []string
	}{
		{
			name:   "defaults are valid",
			mutate: func(*Settings) {},
		},
		{
			name: "an active and a pending model are valid",
			mutate: func(s *Settings) {
				s.ActiveModelKey = "api.voyageai.com/voyage-3/1024"
				s.Dimensions = 1024
				s.PendingModelKey = "localhost:11434/nomic-embed-text/768"
				s.PendingDimensions = 768
			},
		},
		{
			name: "a paused organization says why",
			mutate: func(s *Settings) {
				s.Paused = true
				s.PausedReason = PauseReasonBudget
			},
		},
		{
			name:   "tenant is required",
			mutate: func(s *Settings) { s.OrganizationID = pulid.Nil },
			fields: []string{"organizationId"},
		},
		{
			name:   "a negative budget is refused",
			mutate: func(s *Settings) { s.MonthlyIndexingBudgetUSD = decimal.NewFromInt(-1) },
			fields: []string{"monthlyIndexingBudgetUsd"},
		},
		{
			name: "a budget is in whole cents",
			mutate: func(s *Settings) {
				s.MonthlyIndexingBudgetUSD = decimal.RequireFromString("1.005")
			},
			fields: []string{"monthlyIndexingBudgetUsd"},
		},
		{
			name:   "paused without a reason",
			mutate: func(s *Settings) { s.Paused = true },
			fields: []string{"pausedReason"},
		},
		{
			name:   "a reason without a pause",
			mutate: func(s *Settings) { s.PausedReason = PauseReasonManual },
			fields: []string{"pausedReason"},
		},
		{
			name: "an unknown pause reason",
			mutate: func(s *Settings) {
				s.Paused = true
				s.PausedReason = PauseReason("Tired")
			},
			fields: []string{"pausedReason"},
		},
		{
			name: "an active model needs its dimensions",
			mutate: func(s *Settings) {
				s.ActiveModelKey = "api.openai.com/text-embedding-3-small/1536"
			},
			fields: []string{"dimensions"},
		},
		{
			name:   "dimensions need a model",
			mutate: func(s *Settings) { s.Dimensions = 768 },
			fields: []string{"activeModelKey"},
		},
		{
			name: "only the allowed dimensions",
			mutate: func(s *Settings) {
				s.ActiveModelKey = "api.openai.com/text-embedding-3-large/3072"
				s.Dimensions = 3072
			},
			fields: []string{"dimensions"},
		},
		{
			name: "the pending model differs from the active one",
			mutate: func(s *Settings) {
				s.ActiveModelKey = "api.voyageai.com/voyage-3/1024"
				s.Dimensions = 1024
				s.PendingModelKey = "api.voyageai.com/voyage-3/1024"
				s.PendingDimensions = 1024
			},
			fields: []string{"pendingModelKey"},
		},
		{
			name: "a model key has no surrounding spaces",
			mutate: func(s *Settings) {
				s.ActiveModelKey = " api.voyageai.com/voyage-3/1024"
				s.Dimensions = 1024
			},
			fields: []string{"activeModelKey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			settings := validSettings()
			tt.mutate(settings)

			fields := fieldsWithErrors(settings)
			if len(tt.fields) == 0 {
				assert.Empty(t, fields)
				return
			}
			assert.ElementsMatch(t, tt.fields, fields)
		})
	}
}

func TestSettingsSourcesAndModels(t *testing.T) {
	t.Parallel()

	settings := validSettings()
	settings.DocumentsEnabled = false
	assert.Equal(
		t,
		[]SourceType{SourceTypeMemory, SourceTypeInboundMessage},
		settings.EnabledSourceTypes(),
	)
	assert.Empty(t, settings.IndexedModelKeys())

	settings.ActiveModelKey = "a"
	settings.PendingModelKey = "b"
	assert.Equal(t, []string{"a", "b"}, settings.IndexedModelKeys())
}

func TestUnavailableErrorMatchesTheSentinel(t *testing.T) {
	t.Parallel()

	err := Availability{Reason: UnavailableReasonSchemaMissing, ExtensionInstalled: true}.Err()
	assert.ErrorIs(t, err, ErrVectorUnavailable)
	assert.Contains(t, err.Error(), "SchemaMissing")
	assert.NoError(t, Availability{Available: true}.Err())
	assert.True(t, UnavailableReasonSchemaMissing.IsValid())
}
