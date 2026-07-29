package integration_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/stretchr/testify/assert"
)

func TestHasEnabledTelematics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		records []*integration.Integration
		want    bool
	}{
		{
			name: "enabled samsara integration",
			records: []*integration.Integration{
				{
					Type:     integration.TypeSamsara,
					Category: integration.CategoryTelematics,
					Enabled:  true,
				},
			},
			want: true,
		},
		{
			name: "disabled samsara integration",
			records: []*integration.Integration{
				{
					Type:     integration.TypeSamsara,
					Category: integration.CategoryTelematics,
					Enabled:  false,
				},
			},
			want: false,
		},
		{
			name: "enabled non-telematics category",
			records: []*integration.Integration{
				{
					Type:     integration.TypeGoogleMaps,
					Category: integration.CategoryMappingRouting,
					Enabled:  true,
				},
			},
			want: false,
		},
		{
			name: "telematics category with unsupported type",
			records: []*integration.Integration{
				{
					Type:     integration.TypeHERE,
					Category: integration.CategoryTelematics,
					Enabled:  true,
				},
			},
			want: false,
		},
		{
			name:    "no integrations",
			records: nil,
			want:    false,
		},
		{
			name:    "nil record is skipped",
			records: []*integration.Integration{nil},
			want:    false,
		},
		{
			name: "mixed set with one live provider",
			records: []*integration.Integration{
				{
					Type:     integration.TypeGoogleMaps,
					Category: integration.CategoryMappingRouting,
					Enabled:  true,
				},
				{
					Type:     integration.TypeSamsara,
					Category: integration.CategoryTelematics,
					Enabled:  true,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, integration.HasEnabledTelematics(tt.records))
		})
	}
}

func TestType_SupportsTelematics(t *testing.T) {
	t.Parallel()

	assert.True(t, integration.TypeSamsara.SupportsTelematics())
	assert.False(t, integration.TypeGoogleMaps.SupportsTelematics())
	assert.False(t, integration.TypePCMiler.SupportsTelematics())
}
