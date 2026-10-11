package usageservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/stretchr/testify/require"
)

func TestNoopUsageProvider(t *testing.T) {
	t.Parallel()

	provider := NewNoopUsageProvider()

	limitResult, err := provider.CheckLimit(context.Background(), &platformcatalogports.UsageLimitCheckRequest{
		MeterKey: platformcatalog.MeterAPIRequests,
		Quantity: 1,
	})
	require.NoError(t, err)
	require.True(t, limitResult.Allowed)

	recordResult, err := provider.RecordUsage(context.Background(), &platformcatalogports.UsageRecordRequest{
		MeterKey: platformcatalog.MeterAPIRequests,
		Quantity: 1,
	})
	require.NoError(t, err)
	require.False(t, recordResult.Recorded)
	require.EqualValues(t, 1, recordResult.Quantity)
}
