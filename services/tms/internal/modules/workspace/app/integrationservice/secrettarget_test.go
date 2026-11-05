package integrationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireSecretsForNewTarget(t *testing.T) {
	t.Parallel()

	spec := integration.ConfigSpecs[integration.TypeResend]
	saved := map[string]any{"apiKey": "ciphertext", "baseUrl": "https://api.resend.com"}

	tests := []struct {
		name     string
		incoming map[string]string
		final    map[string]any
		refused  bool
	}{
		{
			name:  "the same address keeps the saved secret",
			final: map[string]any{"baseUrl": "https://api.resend.com/"},
		},
		{
			name:    "a new address needs the secret again",
			final:   map[string]any{"baseUrl": "https://collector.example"},
			refused: true,
		},
		{
			name:     "a new address with the secret entered is fine",
			incoming: map[string]string{"apiKey": "re-entered"},
			final:    map[string]any{"baseUrl": "https://collector.example"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := requireSecretsForNewTarget(spec, tt.incoming, saved, tt.final)

			if !tt.refused {
				require.NoError(t, err)
				return
			}
			var multiErr *errortypes.MultiError
			assert.ErrorAs(t, err, &multiErr)
		})
	}
}
