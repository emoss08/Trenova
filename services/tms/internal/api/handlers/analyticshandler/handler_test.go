package analyticshandler

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/require"
)

func TestValidateAnalyticsRequest_AllowsTomorrowsPickupsInclude(t *testing.T) {
	t.Parallel()

	req := &services.AnaltyicsRequest{
		Page:    services.ShipmentAnalyticsPage,
		Include: includeTomorrowsPickups,
	}

	require.NoError(t, validateAnalyticsRequest(req))
}

func TestValidateAnalyticsRequest_RejectsUnsupportedInclude(t *testing.T) {
	t.Parallel()

	req := &services.AnaltyicsRequest{
		Page:    services.ShipmentAnalyticsPage,
		Include: "unsupported",
	}

	require.Error(t, validateAnalyticsRequest(req))
}
