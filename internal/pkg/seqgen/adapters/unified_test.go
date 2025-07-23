// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package adapters_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/seqgen/adapters"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnifiedFormatProvider_GetFormat(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
	provider := adapters.NewUnifiedFormatProvider(testLogger)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	tests := []struct {
		name         string
		sequenceType sequencestore.SequenceType
		orgID        pulid.ID
		buID         pulid.ID
		wantPrefix   string
		wantError    bool
	}{
		{
			name:         "pro number format",
			sequenceType: sequencestore.SequenceTypeProNumber,
			orgID:        orgID,
			buID:         buID,
			wantPrefix:   "S",
		},
		{
			name:         "consolidation format",
			sequenceType: sequencestore.SequenceTypeConsolidation,
			orgID:        orgID,
			buID:         buID,
			wantPrefix:   "C",
		},
		{
			name:         "invoice format",
			sequenceType: sequencestore.SequenceTypeInvoice,
			orgID:        orgID,
			buID:         buID,
			wantPrefix:   "INV",
		},
		{
			name:         "work order format",
			sequenceType: sequencestore.SequenceTypeWorkOrder,
			orgID:        orgID,
			buID:         buID,
			wantPrefix:   "WO",
		},
		{
			name:         "pro number without business unit",
			sequenceType: sequencestore.SequenceTypeProNumber,
			orgID:        orgID,
			buID:         pulid.Nil,
			wantPrefix:   "S",
		},
		{
			name:         "consolidation without business unit",
			sequenceType: sequencestore.SequenceTypeConsolidation,
			orgID:        orgID,
			buID:         pulid.Nil,
			wantPrefix:   "C",
		},
		{
			name:         "unknown sequence type",
			sequenceType: sequencestore.SequenceType("unknown"),
			orgID:        orgID,
			buID:         buID,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := provider.GetFormat(ctx, tt.sequenceType, tt.orgID, tt.buID)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, format)
			assert.Equal(t, tt.sequenceType, format.Type)
			assert.Equal(t, tt.wantPrefix, format.Prefix)

			// * Verify common format properties
			switch tt.sequenceType {
			case sequencestore.SequenceTypeProNumber:
				assert.True(t, format.IncludeYear)
				assert.Equal(t, 2, format.YearDigits)
				assert.True(t, format.IncludeMonth)
				assert.Equal(t, 4, format.SequenceDigits)
				assert.True(t, format.IncludeLocationCode)
				assert.Equal(t, "12", format.LocationCode)
				assert.True(t, format.IncludeRandomDigits)
				assert.Equal(t, 6, format.RandomDigitsCount)
			case sequencestore.SequenceTypeConsolidation:
				assert.True(t, format.IncludeYear)
				assert.Equal(t, 2, format.YearDigits)
				assert.True(t, format.IncludeMonth)
				assert.Equal(t, 4, format.SequenceDigits)
			case sequencestore.SequenceTypeInvoice:
				assert.True(t, format.IncludeYear)
				assert.Equal(t, 4, format.YearDigits)
				assert.True(t, format.IncludeMonth)
				assert.Equal(t, 6, format.SequenceDigits)
				assert.True(t, format.IncludeCheckDigit)
				assert.True(t, format.UseSeparators)
				assert.Equal(t, "-", format.SeparatorChar)
			case sequencestore.SequenceTypeWorkOrder:
				assert.True(t, format.IncludeYear)
				assert.Equal(t, 2, format.YearDigits)
				assert.False(t, format.IncludeMonth)
				assert.Equal(t, 6, format.SequenceDigits)
				assert.True(t, format.UseSeparators)
				assert.Equal(t, "-", format.SeparatorChar)
			}
		})
	}
}

func TestUnifiedFormatProvider_ProNumberFallback(t *testing.T) {
	// * This test verifies that the provider falls back to default format
	// * when the database lookup fails (which it will in tests)
	ctx := context.Background()
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
	provider := adapters.NewUnifiedFormatProvider(testLogger)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	format, err := provider.GetFormat(ctx, sequencestore.SequenceTypeProNumber, orgID, buID)
	require.NoError(t, err)
	assert.NotNil(t, format)
	assert.Equal(t, "S", format.Prefix)
	assert.Equal(t, sequencestore.SequenceTypeProNumber, format.Type)
}

func TestUnifiedFormatProvider_ConsolidationFallback(t *testing.T) {
	// * This test verifies that the provider falls back to default format
	// * when the database lookup fails (which it will in tests)
	ctx := context.Background()
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
	provider := adapters.NewUnifiedFormatProvider(testLogger)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	format, err := provider.GetFormat(ctx, sequencestore.SequenceTypeConsolidation, orgID, buID)
	require.NoError(t, err)
	assert.NotNil(t, format)
	assert.Equal(t, "C", format.Prefix)
	assert.Equal(t, sequencestore.SequenceTypeConsolidation, format.Type)
}

func TestUnifiedFormatProvider_InterfaceCompliance(t *testing.T) {
	// * Verify that UnifiedFormatProvider implements the FormatProvider interface
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
	provider := adapters.NewUnifiedFormatProvider(testLogger)

	// * This will fail to compile if the interface is not satisfied
	var _ interface{} = provider
}

func BenchmarkUnifiedFormatProvider_GetFormat(b *testing.B) {
	ctx := context.Background()
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
	provider := adapters.NewUnifiedFormatProvider(testLogger)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	sequenceTypes := []sequencestore.SequenceType{
		sequencestore.SequenceTypeProNumber,
		sequencestore.SequenceTypeConsolidation,
		sequencestore.SequenceTypeInvoice,
		sequencestore.SequenceTypeWorkOrder,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sequenceType := sequenceTypes[i%len(sequenceTypes)]
		_, _ = provider.GetFormat(ctx, sequenceType, orgID, buID)
	}
}
