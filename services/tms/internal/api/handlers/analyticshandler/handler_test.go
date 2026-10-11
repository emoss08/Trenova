package analyticshandler

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/require"
)

func TestValidateAnalyticsRequest_AllowsTomorrowsPickupsInclude(t *testing.T) {
	t.Parallel()

	req := &kernelports.AnaltyicsRequest{
		Page:    kernelports.ShipmentAnalyticsPage,
		Include: includeTomorrowsPickups,
	}

	require.NoError(t, validateAnalyticsRequest(req))
}

func TestValidateAnalyticsRequest_RejectsUnsupportedInclude(t *testing.T) {
	t.Parallel()

	req := &kernelports.AnaltyicsRequest{
		Page:    kernelports.ShipmentAnalyticsPage,
		Include: "unsupported",
	}

	require.Error(t, validateAnalyticsRequest(req))
}
