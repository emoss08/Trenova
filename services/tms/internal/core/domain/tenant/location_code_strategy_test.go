package tenant

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocationCodeStrategyValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		strategy *LocationCodeStrategy
		wantErr  string
	}{
		{
			name:     "nil uses defaults",
			strategy: nil,
		},
		{
			name: "valid no separator",
			strategy: &LocationCodeStrategy{
				Components:     []LocationCodeComponent{LocationCodeComponentName},
				ComponentWidth: 3,
				SequenceDigits: 4,
				Separator:      "",
				Casing:         LocationCodeCasingLower,
				FallbackPrefix: "LOC",
			},
		},
		{
			name: "invalid component",
			strategy: &LocationCodeStrategy{
				Components:     []LocationCodeComponent{"county"},
				ComponentWidth: 3,
				SequenceDigits: 3,
				Separator:      "-",
				Casing:         LocationCodeCasingUpper,
				FallbackPrefix: "LOC",
			},
			wantErr: "component",
		},
		{
			name: "invalid separator",
			strategy: &LocationCodeStrategy{
				Components:     []LocationCodeComponent{LocationCodeComponentName},
				ComponentWidth: 3,
				SequenceDigits: 3,
				Separator:      "|",
				Casing:         LocationCodeCasingUpper,
				FallbackPrefix: "LOC",
			},
			wantErr: "separator",
		},
		{
			name: "invalid casing",
			strategy: &LocationCodeStrategy{
				Components:     []LocationCodeComponent{LocationCodeComponentName},
				ComponentWidth: 3,
				SequenceDigits: 3,
				Separator:      "-",
				Casing:         "title",
				FallbackPrefix: "LOC",
			},
			wantErr: "casing",
		},
		{
			name: "empty fallback after normalization",
			strategy: &LocationCodeStrategy{
				Components:     []LocationCodeComponent{LocationCodeComponentName},
				ComponentWidth: 3,
				SequenceDigits: 3,
				Separator:      "-",
				Casing:         LocationCodeCasingUpper,
				FallbackPrefix: "***",
			},
			wantErr: "fallback prefix must contain",
		},
		{
			name: "too long",
			strategy: &LocationCodeStrategy{
				Components: []LocationCodeComponent{
					LocationCodeComponentName,
					LocationCodeComponentCity,
					LocationCodeComponentState,
					LocationCodeComponentPostalCode,
				},
				ComponentWidth: 8,
				SequenceDigits: 3,
				Separator:      "-",
				Casing:         LocationCodeCasingUpper,
				FallbackPrefix: "LOC",
			},
			wantErr: "cannot exceed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.strategy.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
