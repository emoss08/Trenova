package monitoringconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracingConfig_AISamplingRateDefaultsToEverything(t *testing.T) {
	t.Parallel()

	var unset TracingConfig
	assert.InDelta(t, 1.0, unset.GetAISamplingRate(), 0)

	none := 0.0
	assert.InDelta(t, 0.0, (&TracingConfig{AISamplingRate: &none}).GetAISamplingRate(), 0)
}

func TestTracingConfig_TraceURL(t *testing.T) {
	t.Parallel()

	cfg := &TracingConfig{TraceURLTemplate: "https://tempo.example.com/trace/{traceId}?view=1"}
	assert.Equal(t,
		"https://tempo.example.com/trace/4bf92f3577b34da6a3ce929d0e0e4736?view=1",
		cfg.TraceURL("4bf92f3577b34da6a3ce929d0e0e4736"))
	assert.Empty(t, cfg.TraceURL(""))
	assert.Equal(t, "https://tempo.example.com/trace/a%2Fb?view=1", cfg.TraceURL("a/b"))
	assert.Empty(t, (&TracingConfig{}).TraceURL("4bf92f3577b34da6a3ce929d0e0e4736"))
}

func TestValidateTracingConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		wantErr  error
	}{
		{name: "no template", template: ""},
		{name: "https template", template: "https://tempo.example.com/trace/{traceId}"},
		{name: "http template with query", template: "http://jaeger:16686/trace/{traceId}?ui=1"},
		{
			name:     "template without placeholder",
			template: "https://tempo.example.com/trace/",
			wantErr:  ErrTraceURLTemplateMissingPlaceholder,
		},
		{
			name:     "template without scheme",
			template: "tempo.example.com/trace/{traceId}",
			wantErr:  ErrTraceURLTemplateInvalid,
		},
		{
			name:     "template with a script scheme",
			template: "javascript://tempo/{traceId}",
			wantErr:  ErrTraceURLTemplateInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := TracingConfig{TraceURLTemplate: tt.template}

			err := cfg.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
